package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/lib/constants"
	sshutils "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/system"
	"github.com/gravitational/trace"

	log "github.com/sirupsen/logrus"
)

const (
	tfVarsFile           = "robotest.tfvars.json"
	terraformRepeatAfter = time.Second * 5

	azureCloud = "azure"
	awsCloud   = "aws"
)

func New(stateDir string, config Config) (*terraform, error) {
	user, keypath := config.SSHConfig()

	return &terraform{
		Entry: log.WithFields(log.Fields{
			constants.FieldProvisioner: "terraform",
			constants.FieldCluster:     config.ClusterName,
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
		Entry: log.WithFields(log.Fields{
			constants.FieldProvisioner: "terraform",
			constants.FieldCluster:     config.ClusterName,
		}),
		Config:      config,
		stateDir:    stateConfig.Dir,
		installerIP: stateConfig.InstallerAddr,
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
		return nil, trace.Errorf("No Terraform configs at %s", r.ScriptPath)
	}

	// sometimes terraform cannot receive all required params
	// most often public IPs take time to allocate (on Azure)
	for {
		err := r.terraform(ctx)
		if err == nil {
			if withInstaller {
				nodes := r.pool.Nodes()
				if len(nodes) == 0 { // should not happen, and doesn't make sense to retry
					return nil, trace.Errorf("Zero nodes were allocated")
				}
				r.installerIP = nodes[0].Addr()
				return nodes[0], nil
			}
			return nil, nil
		}

		if !trace.IsRetryError(err) {
			return nil, trace.Wrap(err, "Terraform creation failed")
		}
		log.Warningf("terraform experienced transient error %s, will retry in %v",
			err.Error(), terraformRepeatAfter)

		select {
		case <-ctx.Done():
			return nil, trace.Wrap(err, "Terraform creation timed out")
		case <-time.After(terraformRepeatAfter):
		}
	}

}

func (r *terraform) terraform(ctx context.Context) (err error) {
	output, err := r.boot(ctx)
	if err != nil {
		return trace.Wrap(err)
	}

	// find all nodes' private IPs
	match := rePrivateIPs.FindStringSubmatch(output)
	if len(match) != 2 {
		return trace.NotFound(
			"failed to extract private IPs from terraform output: %v", match)
	}
	privateIPs := strings.Split(strings.TrimSpace(match[1]), " ")

	// find all nodes' public IPs
	match = rePublicIPs.FindStringSubmatch(output)
	if len(match) != 2 {
		// one of the reasons is that public IP allocation is incomplete yet
		// which happens for Azure; we will just repeat boot process once again
		return trace.Retry(
			trace.NotFound("failed to extract public IPs from terraform output: %v", match),
			"terraform may not be able to acquire values of every parameter on create")
	}
	publicIPs := strings.Split(strings.TrimSpace(match[1]), " ")

	if len(privateIPs) != len(publicIPs) {
		return trace.BadParameter("number of private IPs is different than public IPs: %v != %v",
			len(privateIPs), len(publicIPs))
	}

	nodes := make([]infra.Node, 0, len(publicIPs))
	for i, addr := range publicIPs {
		nodes = append(nodes, &node{privateIP: privateIPs[i], publicIP: addr, owner: r})
	}
	r.pool = infra.NewNodePool(nodes, nil)

	r.Debugf("cluster: %#v", nodes)
	return nil
}

func (r *terraform) destroyAzure(ctx context.Context) error {
	cfg := r.Config.Azure
	if cfg == nil {
		return trace.Errorf("azure config is nil")
	}

	token, err := AzureGetAuthToken(ctx, AzureAuthParam{
		ClientId:     cfg.ClientId,
		ClientSecret: cfg.ClientSecret,
		TenantId:     cfg.TenantId})
	if err != nil {
		return trace.Wrap(err)
	}

	err = AzureRemoveResourceGroup(ctx, token, cfg.SubscriptionId, cfg.ResourceGroup)
	return trace.Wrap(err)
}

func (r *terraform) Destroy(ctx context.Context) error {
	r.Debugf("destroying terraform cluster: %v", r.stateDir)

	if r.Config.CloudProvider == azureCloud {
		err := r.destroyAzure(ctx)
		if err != nil {
			return trace.Wrap(err, "azureDestroy %v", err)
		}
		err = os.RemoveAll(r.stateDir)
		return trace.Wrap(err, "cleaning up %s: %v", r.stateDir, err)
	}

	varsPath := filepath.Join(r.stateDir, tfVarsFile)
	_, err := r.command(ctx, []string{
		"destroy", "-force",
		"-var", fmt.Sprintf("nodes=%d", r.NumNodes),
		"-var", fmt.Sprintf("os=%s", r.OS),
		fmt.Sprintf("-var-file=%s", varsPath),
	})
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
	cmd, err := r.makeRemoteCommand(r.Config.InstallerURL, "./install")
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
	return fmt.Sprintf("/home/%s/installer/gravity.log", r.sshUser)
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
	}
}

// Write implements io.Writer
func (r *terraform) Write(p []byte) (int, error) {
	fmt.Fprint(os.Stderr, string(p))
	return len(p), nil
}

func (r *terraform) boot(ctx context.Context) (output string, err error) {
	varsPath := filepath.Join(r.stateDir, tfVarsFile)
	err = r.saveVarsJSON(varsPath)
	if err != nil {
		return "", trace.Wrap(err, "failed to store Terraform vars")
	}

	out, err := r.command(ctx, []string{
		"apply", "-input=false",
		"-var", fmt.Sprintf("nodes=%d", r.NumNodes),
		"-var", fmt.Sprintf("os=%s", r.OS),
		fmt.Sprintf("-var-file=%s", varsPath),
	})
	if err != nil {
		return "", trace.Wrap(err, "failed to boot terraform cluster: %s", out)
	}

	return string(out), nil
}

func (r *terraform) command(ctx context.Context, args []string, opts ...system.CommandOptionSetter) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "terraform", args...)
	var out bytes.Buffer
	opts = append(opts, system.Dir(r.stateDir))
	err := system.ExecL(cmd, io.MultiWriter(&out, r), r.Entry, opts...)
	if err != nil {
		return out.Bytes(), trace.Wrap(err, "command %q failed (args %q, wd %q)", cmd.Path, cmd.Args, cmd.Dir)
	}
	return out.Bytes(), nil
}

// serializes terraform vars into given file as JSON
func (r *terraform) saveVarsJSON(varFile string) error {
	var config interface{}
	switch r.Config.CloudProvider {
	case awsCloud:
		config = r.Config.AWS
	case azureCloud:
		config = r.Config.Azure
	default:
		return trace.Errorf("No configuration for cloud %s", r.Config.CloudProvider)
	}

	f, err := os.OpenFile(varFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 440)
	if err != nil {
		return trace.Wrap(err, "Cannot save Terraform Vars file %s", varFile)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent(" ", " ")
	log.Debug(config)
	return trace.Wrap(enc.Encode(config))
}

type terraform struct {
	*log.Entry
	Config

	sshUser, sshKeyPath string
	sshClient           *ssh.Client

	pool        infra.NodePool
	stateDir    string
	installerIP string
}

var (
	rePrivateIPs = regexp.MustCompile("(?m:^ *private_ips *= *([0-9\\. ]+))")
	rePublicIPs  = regexp.MustCompile("(?m:^ *public_ips *= *([0-9\\. ]+))")
)
