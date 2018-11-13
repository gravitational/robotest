package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/providers/azure"
	"github.com/gravitational/robotest/lib/constants"
	sshutils "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/system"

	"github.com/gravitational/trace"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

const (
	tfVarsFile           = "robotest.tfvars.json"
	terraformRepeatAfter = time.Second * 5
)

func New(stateDir string, config Config) (*terraform, error) {
	user, keypath := config.SSHConfig()

	return &terraform{
		FieldLogger: log.WithFields(log.Fields{
			constants.FieldProvisioner: "terraform",
			constants.FieldCluster:     config.ClusterName,
			"state-dir":                stateDir,
		}),
		Config:   config,
		stateDir: stateDir,
		// pool will be reset in Create
		pool: infra.NewNodePool(nil, nil),

		sshUser:    user,
		sshKeyPath: keypath,
	}, nil
}

func NewFromState(config Config, stateConfig infra.ProvisionerState) (*terraform, error) {
	t := &terraform{
		FieldLogger: log.WithFields(log.Fields{
			constants.FieldProvisioner: "terraform",
			constants.FieldCluster:     config.ClusterName,
			"state-dir":                stateConfig.Dir,
		}),
		Config:      config,
		stateDir:    stateConfig.Dir,
		installerIP: stateConfig.InstallerAddr,
	}
	switch specific := stateConfig.Specific.(type) {
	case *State:
		t.loadbalancerIP = specific.LoadBalancerAddr
	}

	t.sshUser, t.sshKeyPath = config.SSHConfig()

	nodes := make([]infra.Node, 0, len(stateConfig.Nodes))
	for _, n := range stateConfig.Nodes {
		nodes = append(nodes, &node{publicIP: n.Addr, owner: t})
	}
	t.pool = infra.NewNodePool(nodes, stateConfig.Allocated)

	return t, nil
}

func (r *terraform) Create(ctx context.Context, withInstaller bool) (installer infra.Node, err error) {
	nfiles, err := system.CopyAll(r.ScriptPath, r.stateDir)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	if nfiles == 0 {
		return nil, trace.NotFound("no terraform configuration at %v", r.ScriptPath)
	}

	// sometimes terraform cannot receive all required params
	// most often public IPs take time to allocate (on Azure)
	for {
		err := r.terraform(ctx)
		if err == nil {
			if withInstaller {
				nodes := r.pool.Nodes()
				if len(nodes) == 0 { // should not happen, and doesn't make sense to retry
					return nil, trace.NotFound("no nodes were allocated")
				}
				if r.installerIP == "" {
					r.installerIP = nodes[0].Addr()
				}
				return nodes[0], nil
			}
			return nil, nil
		}

		if !trace.IsRetryError(err) {
			return nil, trace.Wrap(err, "terraform failed")
		}
		log.WithError(err).Warningf("Terraform experienced transient error, will retry in %v.",
			terraformRepeatAfter)

		select {
		case <-ctx.Done():
			return nil, trace.Wrap(ctx.Err(), "terraform creation timed out")
		case <-time.After(terraformRepeatAfter):
		}
	}
}

// LoadFromExternalState parses terraform output from terraform state file
func (r *terraform) LoadFromExternalState(rdr io.Reader, withInstaller bool) (infra.Node, error) {
	err := r.loadFromState(rdr)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	if withInstaller {
		nodes := r.pool.Nodes()
		if len(nodes) == 0 { // should not happen, and doesn't make sense to retry
			return nil, trace.Errorf("Zero nodes were allocated")
		}
		if r.installerIP == "" {
			r.installerIP = nodes[0].Addr()
		}
		return nodes[0], nil
	}
	return nil, nil
}

func (r *terraform) terraform(ctx context.Context) (err error) {
	f, err := r.boot(ctx)
	if err != nil {
		return trace.Wrap(err)
	}
	defer f.Close()

	err = r.loadFromState(f)
	return trace.Wrap(err)
}

func (r *terraform) loadFromState(rdr io.Reader) error {
	var outputs outputs

	d := json.NewDecoder(rdr)
	if err := d.Decode(&outputs); err != nil {
		return trace.Wrap(err)
	}

	if len(outputs.PublicAddrs.Addrs) == 0 {
		// one of the reasons is that public IP allocation is incomplete yet
		// which happens for Azure; we will just repeat boot process once again
		return trace.NotFound("terraform output contains no public node IPs")
	}

	nodes := make([]infra.Node, 0, len(outputs.PublicAddrs.Addrs))
	for i, addr := range outputs.PublicAddrs.Addrs {
		nodes = append(nodes, &node{
			privateIP: outputs.PrivateAddrs.Addrs[i],
			publicIP:  addr,
			owner:     r,
		})
	}
	r.pool = infra.NewNodePool(nodes, nil)

	r.Debugf("Cluster: %v.", infra.Nodes(nodes))
	return nil
}

func (r *terraform) destroyAzure(ctx context.Context) error {
	cfg := r.Config.Azure
	if cfg == nil {
		return trace.BadParameter("azure config is nil")
	}

	token, err := azure.GetAuthToken(ctx, azure.AuthParam{
		ClientId:     cfg.ClientId,
		ClientSecret: cfg.ClientSecret,
		TenantId:     cfg.TenantId})
	if err != nil {
		return trace.Wrap(err)
	}

	err = azure.RemoveResourceGroup(ctx, *token, cfg.SubscriptionId, cfg.ResourceGroup)
	return trace.Wrap(err)
}

func (r *terraform) Destroy(ctx context.Context) error {
	r.Debugf("Destroying terraform cluster: %v.", r.stateDir)

	if r.Config.CloudProvider == constants.Azure {
		err := r.destroyAzure(ctx)
		if err != nil {
			return trace.Wrap(err, "azureDestroy %v", err)
		}
		err = os.RemoveAll(r.stateDir)
		if err != nil {
			r.Warnf("Failed to clean up: %v", err)
		}
		return nil
	}

	varsPath := filepath.Join(r.stateDir, tfVarsFile)
	destroyCommand := []string{
		"destroy", "-auto-approve",
		"-var", fmt.Sprintf("nodes=%d", r.NumNodes),
		"-var", fmt.Sprintf("os=%s", r.OS),
		fmt.Sprintf("-var-file=%s", varsPath),
	}
	if r.VariablesFile != "" {
		destroyCommand = append(destroyCommand, fmt.Sprintf("-var-file=%s", r.VariablesFile))
	}
	_, err := r.command(ctx, destroyCommand)
	return trace.Wrap(err)
}

func (r *terraform) SelectInterface(installer infra.Node, addrs []string) (int, error) {
	// Fallback to the first available address
	return 0, nil
}

// Connect establishes an SSH connection to the specified address
func (r *terraform) Connect(addrIP string) (*ssh.Session, error) {
	client, err := r.Client(addrIP)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return client.NewSession()
}

// Client establishes an SSH connection to the specified address
func (r *terraform) Client(addrIP string) (*ssh.Client, error) {
	keyFile, err := os.Open(r.sshKeyPath)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return sshutils.Client(fmt.Sprintf("%v:22", addrIP), r.sshUser, keyFile)
}

func (r *terraform) StartInstall(session *ssh.Session) error {
	command := fmt.Sprintf("./gravity install --mode=interactive --log-file=%s", r.InstallerLogPath())
	if r.Config.OnpremProvider {
		command = fmt.Sprintf("%s %s", command, "--cloud-provider=onprem")
	}
	cmd, err := r.makeRemoteCommand(r.Config.InstallerURL, command)
	if err != nil {
		return trace.Wrap(err, "Installer")
	}
	return session.Start(cmd)
}

func (r *terraform) UploadUpdate(session *ssh.Session) error {
	cmd, err := r.makeRemoteCommand(r.Config.InstallerURL, "./upload")
	if err != nil {
		return trace.Wrap(err, "Updater")
	}
	return session.Run(cmd)
}

func (r *terraform) NodePool() infra.NodePool { return r.pool }

func (r *terraform) InstallerLogPath() string {
	return fmt.Sprintf("/home/%s/installer/telekube-system.log", r.sshUser)
}

func (r *terraform) State() infra.ProvisionerState {
	nodes := make([]infra.StateNode, 0, r.pool.Size())
	for _, n := range r.pool.Nodes() {
		nodes = append(nodes, infra.StateNode{Addr: n.(*node).publicIP, KeyPath: r.sshKeyPath})
	}
	allocated := make([]string, 0, r.pool.SizeAllocated())
	for _, node := range r.pool.AllocatedNodes() {
		allocated = append(allocated, node.Addr())
	}
	return infra.ProvisionerState{
		Dir:           r.stateDir,
		InstallerAddr: r.installerIP,
		Nodes:         nodes,
		Allocated:     allocated,
		Specific: &State{
			LoadBalancerAddr: r.loadbalancerIP,
		},
	}
}

func (r *terraform) boot(ctx context.Context) (rc io.ReadCloser, err error) {
	out, err := r.command(ctx, []string{
		"init", "-input=false", "-get-plugins=false",
		fmt.Sprintf("-plugin-dir=%v", constants.TerraformPluginDir),
		r.stateDir},
	)
	if err != nil {
		return nil, trace.Wrap(err, "failed to init terraform: %s", out)
	}

	varsPath := filepath.Join(r.stateDir, tfVarsFile)
	err = r.saveVarsJSON(varsPath)
	if err != nil {
		return nil, trace.Wrap(err, "failed to store terraform vars")
	}

	applyCommand := []string{
		"apply", "-input=false", "-auto-approve",
		"-var", fmt.Sprintf("nodes=%d", r.NumNodes),
		"-var", fmt.Sprintf("os=%s", r.OS),
		fmt.Sprintf("-var-file=%s", varsPath),
	}
	if r.VariablesFile != "" {
		applyCommand = append(applyCommand, fmt.Sprintf("-var-file=%s", r.VariablesFile))
	}

	out, err = r.command(ctx, applyCommand)
	if err != nil {
		return nil, trace.Wrap(err, "failed to boot terraform cluster: %s", out)
	}

	out, err = r.command(ctx, []string{"output", "-json"})
	if err != nil {
		return nil, trace.Wrap(err, "failed to boot terraform cluster: %s", out)
	}
	r.Debug("Cluster outputs:", string(out))

	return ioutil.NopCloser(bytes.NewReader(out)), nil
}

func (r *terraform) command(ctx context.Context, args []string, opts ...system.CommandOptionSetter) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "terraform", args...)
	var out bytes.Buffer
	opts = append(opts,
		system.Dir(r.stateDir),
		system.SetEnv(
			"TF_LOG=DEBUG",
			fmt.Sprintf("TF_LOG_PATH=%v", filepath.Join(r.stateDir, "terraform.log")),
		))
	err := system.ExecL(cmd, &out, r.FieldLogger, opts...)
	r.Infof("Command %#v: %s.", cmd, out.Bytes())
	if err != nil {
		return out.Bytes(), trace.Wrap(err, "command %#v failed: %s", cmd, out.Bytes())
	}
	return out.Bytes(), nil
}

// serializes terraform vars into given file as JSON
func (r *terraform) saveVarsJSON(varFile string) error {
	var config interface{}
	switch r.Config.CloudProvider {
	case constants.AWS:
		config = r.Config.AWS
	case constants.Azure:
		config = r.Config.Azure
	case constants.GCE:
		config = r.Config.GCE
	default:
		return trace.BadParameter("invalid cloud provider: %v", r.Config.CloudProvider)
	}

	f, err := os.OpenFile(varFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, constants.SharedReadWriteMask)
	if err != nil {
		return trace.Wrap(trace.ConvertSystemError(err),
			"failed to save terraform variables file %v", varFile)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent(" ", " ")
	log.Debug(config)
	return trace.Wrap(enc.Encode(config))
}

// MarshalJSON serializes this state object as JSON
func (r *State) MarshalJSON() ([]byte, error) {
	type state State
	bytes, err := json.Marshal((state)(*r))
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return bytes, nil
}

// State defines terraform-specific state
type State struct {
	// LoadBalancerAddr defines the DNS name of the load balancer
	LoadBalancerAddr string `json:"loadbalancer"`
}

// terraform is the terraform-based infrastructure provider
type terraform struct {
	log.FieldLogger
	Config

	sshUser, sshKeyPath string

	pool           infra.NodePool
	stateDir       string
	installerIP    string
	loadbalancerIP string
}

type outputs struct {
	// PodCIDRBlocks specifies the list of CIDR blocks
	// (block per node) dedicated to Pods
	PodCIDRBlocks struct {
		Blocks []string `json:"value"`
	} `json:"pod_cidr_blocks"`
	// ServiceCIDRBlocks specifies the list of CIDR blocks
	// (block per node) dedicated to Services
	ServiceCIDRBlocks struct {
		Blocks []string `json:"value"`
	} `json:"service_cidr_blocks"`
	// PublicAddrs lists public IPs of infrastructure nodes
	PublicAddrs struct {
		Addrs []string `json:"value"`
	} `json:"public_ips"`
	// PrivateAddrs lists private IPs of infrastructure nodes
	PrivateAddrs struct {
		Addrs []string `json:"value"`
	} `json:"private_ips"`
	// LoadBalancerAddr specifies the IP address of the cloud Load Balancer
	LoadBalancerAddr struct {
		Addr string `json:"value"`
	} `json:"load_balancer"`
	// InstallerAddr specifies the IP address of the node selected as installer
	InstallerAddr struct {
		Addr string `json:"value"`
	} `json:"installer_ip"`
}
