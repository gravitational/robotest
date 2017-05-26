package vagrant

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
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
	return &vagrant{
		Entry: log.WithFields(log.Fields{
			constants.FieldProvisioner: "vagrant",
			constants.FieldCluster:     config.ClusterName,
		}),
		stateDir: stateDir,
		// will be reset in Create
		pool:   infra.NewNodePool(nil, nil),
		Config: config,
	}, nil
}

func NewFromState(config Config, stateConfig infra.ProvisionerState) (*vagrant, error) {
	v := &vagrant{
		Entry: log.WithFields(log.Fields{
			constants.FieldProvisioner: "vagrant",
			constants.FieldCluster:     config.ClusterName,
		}),
		stateDir:    stateConfig.Dir,
		installerIP: stateConfig.InstallerAddr,
		Config:      config,
	}
	nodes := make([]infra.Node, 0, len(stateConfig.Nodes))
	for _, n := range stateConfig.Nodes {
		nodes = append(nodes, &node{addrIP: n.Addr, identityFile: n.KeyPath})
	}
	v.pool = infra.NewNodePool(nodes, stateConfig.Allocated)
	return v, nil
}

func (r *vagrant) Create(ctx context.Context, withInstaller bool) (installer infra.Node, err error) {
	file := filepath.Base(r.ScriptPath)
	err = system.CopyFile(r.ScriptPath, filepath.Join(r.stateDir, file))
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = r.boot(withInstaller)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	nodes, err := r.discoverNodes()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	if len(nodes) == 0 {
		return nil, trace.NotFound("failed to discover any nodes")
	}
	if r.Config.NumNodes > len(nodes) {
		return nil, trace.BadParameter("number of requested nodes %v larger than the cluster capacity %v", r.Config.NumNodes, len(nodes))
	}

	r.pool = infra.NewNodePool(nodes, nil)
	r.Debugf("cluster: %#v", r.pool)

	if !withInstaller {
		// No need to pick installer node
		return nil, nil
	}

	// Use first node as installer
	r.installerIP = nodes[0].Addr()
	node, err := r.pool.Node(r.installerIP)

	return node, trace.Wrap(err)
}

func (r *vagrant) Destroy(ctx context.Context) error {
	r.Debugf("destroying vagrant cluster: %v", r.stateDir)
	out, err := r.command(args("destroy", "-f"))
	if err != nil {
		return trace.Wrap(err, "failed to destroy vagrant cluster: %s", out)
	}
	return nil
}

func (r *vagrant) SelectInterface(installer infra.Node, addrs []string) (int, error) {
	for i, addr := range addrs {
		if addr == installer.(*node).addrIP {
			return i, nil
		}
	}
	return -1, trace.NotFound("failed to select installer interface from %v", addrs)
}

// Connect establishes an SSH connection to the specified address
func (r *vagrant) Connect(addrIP string) (*ssh.Session, error) {
	node, err := r.pool.Node(addrIP)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return node.Connect()
}

func (r *vagrant) Client(addrIP string) (*ssh.Client, error) {
	node, err := r.pool.Node(addrIP)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return node.Client()
}

func (r *vagrant) StartInstall(session *ssh.Session) error {
	return session.Start(installerCommand)
}

func (r *vagrant) UploadUpdate(session *ssh.Session) error {
	// upload new installer to all remote nodes
	if err := r.rsyncStateDir(); err != nil {
		return trace.Wrap(err)
	}
	return session.Run(uploadUpdateCommand)
}

func (r *vagrant) NodePool() infra.NodePool {
	return r.pool
}

func (r *vagrant) InstallerLogPath() string {
	return installerLogPath
}

func (r *vagrant) State() infra.ProvisionerState {
	nodes := make([]infra.StateNode, 0, r.pool.Size())
	for _, n := range r.pool.Nodes() {
		nodes = append(nodes, infra.StateNode{Addr: n.(*node).addrIP, KeyPath: n.(*node).identityFile})
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

func (r *vagrant) boot(withInstaller bool) error {
	if withInstaller {
		err := r.syncInstallerTarball()
		if err != nil {
			return trace.Wrap(err)
		}
	}
	out, err := r.command(args("up"), system.SetEnv(
		fmt.Sprintf("VAGRANT_VAGRANTFILE=%v", r.ScriptPath),
		"ROBO_USE_SCRIPTS=true",
	))
	if err != nil {
		return trace.Wrap(err, "failed to provision vagrant cluster: %s", out)
	}
	return nil
}

func (r *vagrant) syncInstallerTarball() error {
	if r.InstallerURL == "" {
		return nil
	}
	target := filepath.Join(r.stateDir, "installer.tar.gz")
	log.Debugf("copy %v -> %v", r.InstallerURL, target)
	err := system.CopyFile(r.InstallerURL, target)
	if err != nil {
		return trace.Wrap(err, "failed to copy installer tarball %q to %q", r.InstallerURL, target)
	}
	return nil
}

func (r *vagrant) rsyncStateDir() (err error) {
	err = r.syncInstallerTarball()
	if err != nil {
		return trace.Wrap(err)
	}

	_, err = r.command(args("rsync"))
	if err != nil {
		return trace.Wrap(err, "failed to rsync state folder to remote machine")
	}
	return nil
}

func (r *vagrant) discoverNodes() ([]infra.Node, error) {
	out, err := r.command(args("ssh-config"))
	if err != nil {
		return nil, trace.Wrap(err, "failed to query SSH config: %s", out)
	}

	nodes, err := parseSSHConfig(out, r.getIPLibvirt)
	if err != nil {
		return nil, trace.Wrap(err, "failed to parse SSH config")
	}

	return nodes, nil
}

func (r *vagrant) getIPLibvirt(nodename string) (string, error) {
	var out bytes.Buffer
	cmd := exec.Command("virsh", "list", "--name")
	err := system.ExecL(cmd, io.MultiWriter(&out, r), r.Entry)
	if err != nil {
		return "", trace.Wrap(err, "failed to discover VM public IP: %s", out)
	}
	var domainName string
	for _, name := range strings.Split(out.String(), "\n") {
		if strings.Contains(name, nodename) {
			domainName = name
			break
		}
	}
	if domainName == "" {
		return "", trace.NotFound("failed to find libvirt domain for node %q: %s", nodename)
	}

	out.Reset()
	cmd = exec.Command("virsh", "dumpxml", domainName)
	err = system.ExecL(cmd, io.MultiWriter(&out, r), r.Entry)
	if err != nil {
		return "", trace.Wrap(err, "failed to discover VM public IP: %s", out.Bytes())
	}

	var domain domain
	err = xml.Unmarshal(out.Bytes(), &domain)
	if err != nil {
		return "", trace.Wrap(err, "failed to discover VM public IP: %s", out.Bytes())
	}

	var macAddr string
	for _, iface := range domain.Devices.Interfaces {
		if iface.Type == "network" && iface.Source.Network == "vagrant-libvirt" {
			macAddr = iface.Mac.Addr
			break
		}
	}
	if macAddr == "" {
		return "", trace.NotFound("failed to find MAC address for node %q: %s", nodename)
	}

	arpFile, err := os.Open("/proc/net/arp")
	if err != nil {
		return "", trace.Wrap(err, "failed to read arp table")
	}
	defer arpFile.Close()

	s := bufio.NewScanner(arpFile)
	// Skip the header
	if !s.Scan() {
		return "", trace.Wrap(err, "failed to read arp table")
	}
	for s.Scan() {
		line := s.Text()
		fields := strings.Fields(line)
		if len(fields) != 6 {
			r.Warningf("skipping invalid arp entry %q", line)
			continue
		}
		if macAddr == fields[3] {
			return fields[0], nil
		}
	}
	if err := s.Err(); err != nil {
		return "", trace.Wrap(err, "failed to read arp table")
	}

	return "", trace.NotFound("failed to find IP address for node %q", nodename)
}

func (r *vagrant) getIPVirtualbox(nodename string) (string, error) {
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
	opts = append(opts, system.Dir(r.stateDir), system.SetEnv(fmt.Sprintf("ROBO_NUM_NODES=%v", r.Config.NumNodes)))
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

func (r *node) PrivateAddr() string {
	return r.addrIP
}

func (r *node) Connect() (*ssh.Session, error) {
	client, err := r.Client()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return session, nil
}

func (r *node) Client() (*ssh.Client, error) {
	keyFile, err := os.Open(r.identityFile)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer keyFile.Close()
	return sshutils.Client(fmt.Sprintf("%v:22", r.addrIP), "vagrant", keyFile)
}

func (r node) String() string {
	return fmt.Sprintf("node(addr=%v)", r.addrIP)
}

func args(opts ...string) (result []string) {
	return opts
}

func parseSSHConfig(config []byte, getIP func(string) (string, error)) (nodes []infra.Node, err error) {
	s := bufio.NewScanner(bytes.NewReader(config))
	var host string
	// nodes maps node IP address to node
	for s.Scan() {
		line := s.Text()
		switch {
		case strings.HasPrefix(line, "Host"):
			// Start a new node
			host = strings.TrimSpace(strings.TrimPrefix(line, "Host"))
		case strings.HasPrefix(line, "  IdentityFile"):
			path := strings.TrimSpace(strings.TrimPrefix(line, "  IdentityFile"))
			identityFile, err := strconv.Unquote(path)
			if err != nil {
				identityFile = path
			}
			addrIP, err := getIP(host)
			if err != nil {
				return nil, trace.Wrap(err, "failed to determine IP address of the host %q", host)
			}
			nodes = append(nodes, &node{addrIP: addrIP, identityFile: identityFile})
		}
	}
	return nodes, nil
}

type vagrant struct {
	*log.Entry
	Config

	pool        infra.NodePool
	stateDir    string
	installerIP string
	// nodes maps node address to a node.
	// nodes represents the total cluster capacity as defined by the Vagrantfile
	nodes map[string]node
}

type node struct {
	identityFile string
	addrIP       string
}

type domain struct {
	Devices struct {
		Interfaces []struct {
			Type string `xml:"type,attr"`
			Mac  struct {
				Addr string `xml:"address,attr"`
			} `xml:"mac"`
			Source struct {
				Network string `xml:"network,attr"`
			} `xml:"source"`
		} `xml:"interface"`
	} `xml:"devices"`
}

const installerCommand = `
mkdir -p /home/vagrant/installer; \
tar -xvf /vagrant/installer.tar.gz -C /home/vagrant/installer; \
/home/vagrant/installer/install`

const uploadUpdateCommand = `
rm -rf /home/vagrant/installer; mkdir -p /home/vagrant/installer; \
tar -xvf /vagrant/installer.tar.gz -C /home/vagrant/installer; \
cd /home/vagrant/installer/; sudo ./upload`

const installerLogPath = "/home/vagrant/installer/gravity.log"
