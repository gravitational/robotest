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
	}, nil
}

func NewFromState(config Config, stateConfig infra.ProvisionerState) (*terraform, error) {
	// TODO
	return nil, nil
}

func (r *terraform) Create() (installer infra.Node, err error) {
	file := filepath.Base(r.ScriptPath)
	err = system.CopyFile(r.ScriptPath, filepath.Join(r.stateDir, file))
	if err != nil {
		return nil, trace.Wrap(err)
	}

	output, err := r.boot()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	// find installer public IP
	match := reInstallerIP.FindStringSubmatch(output)
	if len(match) != 2 {
		return nil, trace.NotFound(
			"failed to extract installer IP from terraform output: %v", match)
	}
	installerIP := strings.TrimSpace(match[1])

	// find all nodes' private IPs
	match = rePrivateIPs.FindStringSubmatch(output)
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

	r.nodes = make(map[string]node)
	for i, addr := range publicIPs {
		r.nodes[addr] = node{privateIP: privateIPs[i], publicIP: addr, owner: r}
	}

	r.active = make(map[string]struct{})
	for _, addr := range publicIPs[:r.NumInstallNodes] {
		r.active[addr] = struct{}{}
	}

	r.Debugf("cluster: %#v", r.nodes)
	r.Debugf("install subset: %#v", r.active)

	node := r.nodes[installerIP]
	return &node, nil
}

func (r *terraform) Destroy() error {
	r.Debugf("destroying terraform cluster: %v", r.stateDir)
	args := append([]string{"destroy", "-force"}, getVars(r.Config)...)
	_, err := r.command(args)
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
	return sshutils.Connect(fmt.Sprintf("%v:22", addrIP), "centos", keyFile)
}

func (r *terraform) StartInstall(session *ssh.Session) error {
	return session.Start(installerCommand)
}

func (r *terraform) AllNodes() (nodes []infra.Node) {
	nodes = make([]infra.Node, 0, len(r.nodes))
	for addr := range r.nodes {
		node := r.nodes[addr]
		nodes = append(nodes, &node)
	}
	return nodes
}

func (r *terraform) Nodes() (nodes []infra.Node) {
	nodes = make([]infra.Node, 0, len(r.active))
	for addr := range r.active {
		node := r.nodes[addr]
		nodes = append(nodes, &node)
	}
	return nodes
}

func (r *terraform) NumNodes() int {
	return len(r.active)
}

func (r *terraform) Node(addr string) (infra.Node, error) {
	if node, exists := r.nodes[addr]; exists {
		return &node, nil
	}
	return nil, trace.NotFound("node %q not found", addr)
}

func (r *terraform) Allocate() (infra.Node, error) {
	for _, node := range r.nodes {
		if _, active := r.active[node.publicIP]; !active {
			r.active[node.publicIP] = struct{}{}
			return &node, nil
		}
	}
	return nil, trace.NotFound("cannot allocate node")
}

func (r *terraform) Deallocate(n infra.Node) error {
	delete(r.active, n.(*node).publicIP)
	return nil
}

func (r *terraform) InstallerLogPath() string {
	return installerLogPath
}

func (r *terraform) StateDir() string { return r.stateDir }

func (r *terraform) State() infra.ProvisionerState {
	return infra.ProvisionerState{}
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

func (r *terraform) boot() (output string, err error) {
	args := append([]string{"apply"}, getVars(r.Config)...)
	out, err := r.command(args)
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
		"access_key":    config.AccessKey,
		"secret_key":    config.SecretKey,
		"region":        config.Region,
		"key_pair":      config.KeyPair,
		"instance_type": config.InstanceType,
		"cluster_name":  config.ClusterName,
		"installer_url": config.InstallerURL,
	}
	variables["nodes"] = strconv.Itoa(config.NumNodes)
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

	stateDir string
	nodes    map[string]node
	active   map[string]struct{}
}

type node struct {
	owner     *terraform
	publicIP  string
	privateIP string
}

var (
	reInstallerIP = regexp.MustCompile("(?m:^installer_ip = ([0-9\\.]+))")
	rePrivateIPs  = regexp.MustCompile("(?m:^private_ips = ([0-9\\. ]+))")
	rePublicIPs   = regexp.MustCompile("(?m:^public_ips = ([0-9\\. ]+))")
)

// installerCommand waits for the installer tarball to download, unpacks it and launches the installation
const installerCommand = `while [ ! -f /home/centos/installer.tar.gz ]; do sleep 5; done; \
tar -xvf /home/centos/installer.tar.gz -C /home/centos/installer; \
/home/centos/installer/install`

const installerLogPath = "/home/centos/installer/gravity.log"
