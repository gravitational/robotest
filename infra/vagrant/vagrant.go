package vagrant

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

func New(stateDir string, config Config) (*vagrant, error) {
	err := config.Validate()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return &vagrant{
		Entry: log.WithFields(log.Fields{
			constants.FieldProvisioner: "vagrant",
			constants.FieldCluster:     config.ClusterName,
		}),
		stateDir: stateDir,
		Config:   config,
	}, nil
}

func (r *vagrant) Create() (*infra.ProvisionerOutput, error) {
	file := filepath.Base(r.ScriptPath)
	err := system.CopyFile(r.ScriptPath, filepath.Join(r.stateDir, file))
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = r.boot()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = r.discoverNodes()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	if len(r.nodes) == 0 {
		return nil, trace.NotFound("failed to discover any nodes")
	}
	if r.Config.Nodes > len(r.nodes) {
		return nil, trace.BadParameter("number of requested nodes %v larger than the cluster capacity %v", r.Config.Nodes, len(r.nodes))
	}

	activeNodes := r.Config.Nodes
	if r.Config.Nodes == 0 {
		activeNodes = len(r.nodes)
	}

	publicIPs := make([]string, 0, len(r.nodes))
	for _, node := range r.nodes {
		publicIPs = append(publicIPs, node.addrIP)
	}
	r.active = make(map[string]struct{})
	for _, addr := range publicIPs[:activeNodes] {
		r.active[addr] = struct{}{}
	}

	r.Infof("cluster: %#v", r.nodes)
	r.Infof("install subset: %#v", r.active)

	return &infra.ProvisionerOutput{
		InstallerIP: publicIPs[0],
		PublicIPs:   publicIPs,
	}, nil
}

func (r *vagrant) Destroy() error {
	out, err := r.command(args("destroy", "-f"))
	if err != nil {
		return trace.Wrap(err, "failed to destroy vagrant cluster: %s", out)
	}
	return nil
}

func (r *vagrant) SelectInterface(output infra.ProvisionerOutput, addrs []string) (int, error) {
	for i, addr := range addrs {
		if addr == output.InstallerIP {
			return i, nil
		}
	}
	return -1, trace.NotFound("failed to select installer interface from %v", addrs)
}

// Connect establishes an SSH connection to the specified address
func (r *vagrant) Connect(addrIP string) (*ssh.Session, error) {
	node, ok := r.nodes[addrIP]
	if !ok {
		return nil, trace.NotFound("no node with IP %q", addrIP)
	}
	return node.Connect()
}

func (r *vagrant) StartInstall(session *ssh.Session) error {
	file := filepath.Base(r.InstallerURL)
	target := filepath.Join(r.stateDir, file)
	err := system.CopyFile(r.InstallerURL, target)
	if err != nil {
		return trace.Wrap(err, "failed to copy installer tarball %q to %q", r.InstallerURL, target)
	}
	return session.Start(installerCommand)
}

func (r *vagrant) Nodes() (nodes []infra.Node) {
	nodes = make([]infra.Node, 0, len(r.active))
	for addr := range r.active {
		node := r.nodes[addr]
		nodes = append(nodes, &node)
	}
	return nodes
}

func (r *vagrant) NumNodes() int {
	return len(r.active)
}

func (r *vagrant) Allocate() (infra.Node, error) {
	for _, node := range r.nodes {
		if _, active := r.active[node.addrIP]; !active {
			r.active[node.addrIP] = struct{}{}
			return &node, nil
		}
	}
	return nil, trace.NotFound("cannot allocate node")
}

func (r *vagrant) Deallocate(n infra.Node) error {
	delete(r.active, n.(*node).addrIP)
	return nil
}

func (r *vagrant) boot() error {
	out, err := r.command(args("up"), setEnv(fmt.Sprintf("VAGRANT_VAGRANTFILE=%v", r.ScriptPath)))
	if err != nil {
		return trace.Wrap(err, "failed to provision vagrant cluster: %s", out)
	}
	return nil
}

func (r *vagrant) discoverNodes() error {
	if len(r.nodes) > 0 {
		return nil
	}
	out, err := r.command(args("ssh-config"))
	if err != nil {
		return trace.Wrap(err, "failed to discover SSH key path: %s", out)
	}

	s := bufio.NewScanner(bytes.NewReader(out))
	var host string
	// nodes maps hostname to node
	nodes := make(map[string]node)
	for s.Scan() {
		line := s.Text()
		switch {
		case strings.HasPrefix(line, "Host"):
			// Start a new node
			host = strings.TrimSpace(strings.TrimPrefix(line, "Host"))
		case strings.HasPrefix(line, "  IdentityFile"):
			path, _ := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(line, "  IdentityFile")))
			addrIP, err := r.getIP(host)
			if err != nil {
				return trace.Wrap(err, "failed to determine IP address of the host %q", host)
			}
			node := nodes[addrIP]
			node.identityFile = path
			node.addrIP = addrIP
			nodes[addrIP] = node
		}
	}

	r.nodes = nodes

	return nil
}

func (r *vagrant) getIP(nodename string) (string, error) {
	out, err := r.command(args("ssh", nodename, "-c", "ip r|tail -n 1|cut -d' ' -f12", "--", "-q"))
	if err != nil {
		return "", trace.Wrap(err, "failed to discover VM public IP: %s", out)
	}

	publicIP := strings.TrimSpace(string(out))
	if publicIP == "" {
		return "", trace.NotFound("failed to discover VM public IP")
	}

	return publicIP, nil
}

func (r *vagrant) command(args []string, opts ...system.CommandOptionSetter) ([]byte, error) {
	cmd := exec.Command("vagrant", args...)
	var out bytes.Buffer
	opts = append(opts, system.Dir(r.stateDir))
	err := system.ExecL(cmd, io.MultiWriter(&out, r), r.Entry, opts...)
	if err != nil {
		return out.Bytes(), trace.Wrap(err, "command %q failed (args %q, wd %q)", cmd.Path, cmd.Args, cmd.Dir)
	}
	return out.Bytes(), nil
}

// Write implements io.Writer
func (r *vagrant) Write(p []byte) (int, error) {
	fmt.Fprint(os.Stderr, string(p))
	return len(p), nil
}

func (r *node) Addr() string {
	return r.addrIP
}

func (r *node) Connect() (*ssh.Session, error) {
	keyFile, err := os.Open(r.identityFile)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer keyFile.Close()
	return sshutils.Connect(fmt.Sprintf("%v:22", r.addrIP), "vagrant", keyFile)
}

func args(opts ...string) (result []string) {
	return opts
}

func setEnv(envs ...string) system.CommandOptionSetter {
	return func(cmd *exec.Cmd) {
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, envs...)
	}
}

type vagrant struct {
	*log.Entry
	Config

	stateDir string
	keyPath  string
	// nodes maps node address to a node.
	// nodes represents the total cluster capacity as defined
	// by the Vagrantfile
	nodes map[string]node
	// active defines the subset of nodes forming the current cluster
	// which maybe less than the total capacity (len(nodes))
	active map[string]struct{}
}

type node struct {
	identityFile string
	addrIP       string
}

const installerCommand = `
mkdir -p /home/vagrant/installer; \
tar -xvf /vagrant/installer.tar.gz -C /home/vagrant/installer; \
/home/vagrant/installer/install`
