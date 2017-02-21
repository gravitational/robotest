package terraform

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/lib/constants"
	sshutils "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/system"
	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
)

func New(stateDir string, config Config) (*terraform, error) {
	return &terraform{
		Entry: log.WithFields(log.Fields{
			constants.FieldProvisioner: "terraform",
			constants.FieldCluster:     config.ClusterName,
		}),
		Config:   config,
		stateDir: stateDir,
		// pool will be reset in Create
		pool: infra.NewNodePool(nil, nil),
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

	if config.SSHKeyPath != "" {
		t.Config.SSHKeyPath = config.SSHKeyPath
	}

	if config.SSHKeyPath == "" && len(stateConfig.Nodes) != 0 {
		t.Config.SSHKeyPath = stateConfig.Nodes[0].KeyPath
	}

	nodes := make([]infra.Node, 0, len(stateConfig.Nodes))
	for _, n := range stateConfig.Nodes {
		nodes = append(nodes, &node{publicIP: n.Addr, owner: t})
	}
	t.pool = infra.NewNodePool(nodes, stateConfig.Allocated)

	return t, nil
}

func (r *terraform) Create(withInstaller bool) (installer infra.Node, err error) {
	err = system.CopyFile(filepath.Join(r.stateDir, "terraform.tf"), r.ScriptPath)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	output, err := r.boot()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	if withInstaller {
		// find installer public IP
		match := reInstallerIP.FindStringSubmatch(output)
		if len(match) != 2 {
			return nil, trace.NotFound(
				"failed to extract installer IP from terraform output: %v", match)
		}
		r.installerIP = strings.TrimSpace(match[1])
	}

	// find all nodes' private IPs
	match := rePrivateIPs.FindStringSubmatch(output)
	if len(match) != 2 {
		return nil, trace.NotFound(
			"failed to extract private IPs from terraform output: %v", match)
	}
	privateIPs := strings.Split(strings.TrimSpace(match[1]), " ")

	// find all nodes' public IPs
	match = rePublicIPs.FindStringSubmatch(output)
	if len(match) != 2 {
		return nil, trace.NotFound(
			"failed to extract public IPs from terraform output: %v", match)
	}
	publicIPs := strings.Split(strings.TrimSpace(match[1]), " ")

	if len(privateIPs) != len(publicIPs) {
		return nil, trace.BadParameter(
			"number of private IPs is different than public IPs: %v != %v", len(privateIPs), len(publicIPs))
	}

	nodes := make([]infra.Node, 0, len(publicIPs))
	for i, addr := range publicIPs {
		nodes = append(nodes, &node{privateIP: privateIPs[i], publicIP: addr, owner: r})
	}
	r.pool = infra.NewNodePool(nodes, nil)

	r.Debugf("cluster: %#v", nodes)

	if r.installerIP != "" {
		node, err := r.pool.Node(r.installerIP)
		if err != nil {
			return nil, trace.Wrap(err)
		}
		return node, nil
	}
	return nil, nil
}

func (r *terraform) Destroy() error {
	r.Debugf("destroying terraform cluster: %v", r.stateDir)
	// Pass secrets via environment variables
	accessKeyVar := fmt.Sprintf("TF_VAR_access_key=%v", r.Config.AccessKey)
	secretKeyVar := fmt.Sprintf("TF_VAR_secret_key=%v", r.Config.SecretKey)
	args := append([]string{"destroy", "-force"}, getVars(r.Config)...)
	_, err := r.command(args, system.SetEnv(accessKeyVar, secretKeyVar))
	return trace.Wrap(err)
}

func (r *terraform) SelectInterface(installer infra.Node, addrs []string) (int, error) {
	// Fallback to the first available address
	return 0, nil
}

// Connect establishes an SSH connection to the specified address
func (r *terraform) Connect(addrIP string) (*ssh.Session, error) {
	keyFile, err := os.Open(r.SSHKeyPath)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return sshutils.Connect(fmt.Sprintf("%v:22", addrIP), r.Config.SSHUser, keyFile)
}

func (r *terraform) StartInstall(session *ssh.Session) error {
	return session.Start(installerCommand(r.Config.SSHUser))
}

func (r *terraform) UploadUpdate(session *ssh.Session) error {
	return session.Start(uploadUpdateCommand(r.Config.SSHUser))
}

func (r *terraform) NodePool() infra.NodePool { return r.pool }

func (r *terraform) InstallerLogPath() string {
	return fmt.Sprintf("/home/%s/installer/gravity.log", r.Config.SSHUser)
}

func (r *terraform) State() infra.ProvisionerState {
	nodes := make([]infra.StateNode, 0, r.pool.Size())
	for _, n := range r.pool.Nodes() {
		nodes = append(nodes, infra.StateNode{Addr: n.(*node).publicIP, KeyPath: r.Config.SSHKeyPath})
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

func (r *node) Addr() string {
	return r.publicIP
}

func (r *node) Connect() (*ssh.Session, error) {
	return r.owner.Connect(r.publicIP)
}

func (r node) String() string {
	return fmt.Sprintf("node(addr=%v)", r.publicIP)
}

func (r *terraform) boot() (output string, err error) {
	args := append([]string{"apply"}, getVars(r.Config)...)
	// Pass secrets via environment variables
	accessKeyVar := fmt.Sprintf("TF_VAR_access_key=%v", r.Config.AccessKey)
	secretKeyVar := fmt.Sprintf("TF_VAR_secret_key=%v", r.Config.SecretKey)
	out, err := r.command(args, system.SetEnv(accessKeyVar, secretKeyVar))
	if err != nil {
		return "", trace.Wrap(err, "failed to boot terraform cluster: %s", out)
	}

	return string(out), nil
}

func (r *terraform) command(args []string, opts ...system.CommandOptionSetter) ([]byte, error) {
	cmd := exec.Command("terraform", args...)
	var out bytes.Buffer
	opts = append(opts, system.Dir(r.stateDir))
	err := system.ExecL(cmd, io.MultiWriter(&out, r), r.Entry, opts...)
	if err != nil {
		return out.Bytes(), trace.Wrap(err, "command %q failed (args %q, wd %q)", cmd.Path, cmd.Args, cmd.Dir)
	}
	return out.Bytes(), nil
}

// getVars returns a list of variables to provide to terraform apply/destroy commands
// extracted from the config
func getVars(config Config) []string {
	variables := map[string]string{
		"region":        config.Region,
		"key_pair":      config.KeyPair,
		"instance_type": config.InstanceType,
		"cluster_name":  config.ClusterName,
		"nodes":         strconv.Itoa(config.NumNodes),
		"ssh_user":      config.SSHUser,
	}
	if config.InstallerURL != "" {
		variables["installer_url"] = config.InstallerURL
	}
	var args []string
	for k, v := range variables {
		if strings.TrimSpace(v) != "" {
			args = append(args, "-var", fmt.Sprintf("%v=%v", k, v))
		}
	}
	return args
}

type terraform struct {
	*log.Entry
	Config

	pool        infra.NodePool
	stateDir    string
	installerIP string
}

type node struct {
	owner     *terraform
	publicIP  string
	privateIP string
}

var (
	reInstallerIP = regexp.MustCompile("(?m:^ *installer_ip *= *([0-9\\.]+))")
	rePrivateIPs  = regexp.MustCompile("(?m:^ *private_ips *= *([0-9\\. ]+))")
	rePublicIPs   = regexp.MustCompile("(?m:^ *public_ips *= *([0-9\\. ]+))")
)

// installerCommand returns a shell command to fetch installer tarball, unpack it and launch the installation
func installerCommand(username string) string {
	return fmt.Sprintf(`while [ ! -f /home/%[1]s/installer.tar.gz ]; do sleep 5; done; \
                        tar -xf /home/%[1]s/installer.tar.gz -C /home/%[1]s/installer; \
                        /home/%[1]s/installer/install`, username)
}

// uploadUpdateCommand returns a shell command to fetch installer tarball, unpack it and launch
// uploading new version of application
func uploadUpdateCommand(username string) string {
	return fmt.Sprintf(`while [ ! -f /home/%[1]s/installer.tar.gz ]; do sleep 5; done; \
                        rm -rf /home/%[1]s/installer/*; \
                        tar -xf /home/%[1]s/installer.tar.gz -C /home/%[1]s/installer; \
                        cd /home/%[1]s/installer/; sudo ./upload`, username)
}
