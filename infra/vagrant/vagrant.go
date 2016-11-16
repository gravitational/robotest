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
		Entry: log.WithFields(log.Fields{"provisioner": "vagrant", "cluster": config.ClusterName}),

		stateDir: stateDir,
		config:   config,
	}, nil
}

func (r *vagrant) Create() (*infra.ProvisionerOutput, error) {
	file := filepath.Base(r.config.ScriptPath)
	err := system.CopyFile(r.config.ScriptPath, filepath.Join(r.stateDir, file))
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
		return nil, trace.NotFound("failed to discover any node")
	}

	publicIPs := make([]string, 0, len(r.nodes))
	for _, node := range r.nodes {
		publicIPs = append(publicIPs, node.addrIP)
	}

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
	file := filepath.Base(r.config.InstallerURL)
	target := filepath.Join(r.stateDir, file)
	err := system.CopyFile(r.config.InstallerURL, target)
	if err != nil {
		return trace.Wrap(err, "failed to copy installer tarball %q to %q", r.config.InstallerURL, target)
	}
	return session.Start(installerCommand)
}

func (r *vagrant) Nodes() (nodes []infra.Node) {
	nodes = make([]infra.Node, 0, len(r.nodes))
	// TODO: return only allocated nodes
	for _, node := range r.nodes {
		nodes = append(nodes, &node)
	}
	return nodes
}

func (r *vagrant) Allocate() (infra.Node, error) {
	// TODO
	return nil, nil
}

func (r *vagrant) Deallocate(infra.Node) error {
	// TODO: put node back to node pool
	return nil
}

func (r *vagrant) boot() error {
	out, err := r.command(args("up"), setEnv(fmt.Sprintf("VAGRANT_VAGRANTFILE=%v", r.config.ScriptPath)))
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
		log.Infof("parsing %q", line)
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
	log.Infof("discovered nodes: %#v", nodes)
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

func (r *vagrant) Write(p []byte) (int, error) {
	fmt.Fprint(os.Stderr, string(p))
	return len(p), nil
}

func (r *node) Run(command string, w io.Writer) error {
	session, err := r.Connect()
	if err != nil {
		return trace.Wrap(err)
	}
	return sshutils.RunCommandWithOutput(session, command, w)
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
	result = make([]string, 0, len(opts))
	for _, opt := range opts {
		result = append(result, opt)
	}
	return result
}

func setEnv(envs ...string) system.CommandOptionSetter {
	return func(cmd *exec.Cmd) {
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, envs...)
	}
}

type vagrant struct {
	*log.Entry

	stateDir string
	keyPath  string
	config   Config
	// nodes maps node address to a node
	nodes map[string]node
}

type node struct {
	identityFile string
	addrIP       string
}

const installerCommand = `
mkdir -p /home/vagrant/installer; \
tar -xvf /vagrant/installer.tar.gz -C /home/vagrant/installer; \
/home/vagrant/installer/install`
