package gravity

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"path/filepath"
	"text/template"
	"time"

	"github.com/gravitational/robotest/infra"
	sshutils "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/trace"

	"golang.org/x/crypto/ssh"
)

// Gravity is interface to remote gravity CLI
type Gravity interface {
	// Install operates on initial master node
	Install(ctx context.Context, param InstallCmd) error
	// Status retrieves status
	Status(ctx context.Context) (*GravityStatus, error)
	// SiteReport retrieves site report
	SiteReport(ctx context.Context) error
	// OfflineUpdate tries to upgrade application version
	OfflineUpdate(ctx context.Context, installerUrl string) error
	// Join asks to join existing cluster (or installation in progress)
	Join(ctx context.Context, param JoinCmd) error
	// Leave requests current node leave a cluster
	Leave(ctx context.Context, graceful bool) error
	// Remove requests cluster to evict a given node
	Remove(ctx context.Context, node string, graceful bool) error
	// Uninstall will wipe gravity installation from node
	Uninstall(ctx context.Context) error
	// PowerOff will power off the node
	PowerOff(ctx context.Context, graceful bool) error
	// Reboot will reboot this node and wait until it will become available again
	Reboot(ctx context.Context, graceful bool) error
	// CollectLogs will pull essential logs from node and store it in state dir under node-logs/prefix
	CollectLogs(ctx context.Context, prefix string) (localPath string, err error)
	// Node returns underlying VM instance
	Node() infra.Node
	// Client returns SSH client to VM instance
	Client() *ssh.Client
	// Text representation
	String() string
	// Will log using extended info such as current tag, node info, etc
	Logf(format string, args ...interface{})
}

const (
	// Force an operation
	Force = false
	// Graceful completion of operation
	Graceful = true
)

// InstallCmd install parameters passed to first node
type InstallCmd struct {
	// Token is required to join cluster
	Token string
	// Cluster is Optional name of the cluster. Autogenerated if not set.
	Cluster string
	// Flavor is (Optional) Application flavor. See Application Manifest for details.
	Flavor string
	// K8SConfig is (Optional) File with Kubernetes resources to create in the cluster during installation.
	K8SConfig string
	// PodNetworkCidr is (Optional) CIDR range Kubernetes will be allocating node subnets and pod IPs from. Must be a minimum of /16 so Kubernetes is able to allocate /24 to each node. Defaults to 10.244.0.0/16.
	PodNetworkCIDR string
	// ServiceCidr (Optional) CIDR range Kubernetes will be allocating service IPs from. Defaults to 10.100.0.0/16.
	ServiceCIDR string
}

// JoinCmd represents various parameters for Join
type JoinCmd struct {
	// InstallDir is set automatically
	InstallDir string
	// PeerAddr is other node (i.e. master)
	PeerAddr string
	Token    string
	Role     string
}

// GravityStatus is serialized form of `gravity status` CLI.
type GravityStatus struct {
	Application string
	Cluster     string
	Status      string
	// Token is secure token which prevents rogue nodes from joining the cluster during installation.
	Token string `validation:"required"`
	// Nodes defines nodes the cluster observes
	Nodes []string
}

type gravity struct {
	logFn utils.LogFnType
	node  infra.Node
	ssh   *ssh.Client
	param cloudDynamicParams
	ts    time.Time
}

const (
	retrySSH    = time.Second * 10
	deadlineSSH = time.Minute * 5 // abort if we can't get it within this reasonable period
)

// waits for SSH to be up on node and returns client
func sshClient(baseContext context.Context, logFn utils.LogFnType, node infra.Node) (*ssh.Client, error) {
	ctx, cancel := context.WithTimeout(baseContext, deadlineSSH)
	defer cancel()

	for {
		client, err := node.Client()

		if err == nil {
			return client, nil
		}

		logFn("waiting for SSH on %s, retry in %v, error was %v", node.Addr(), retrySSH, err)
		select {
		case <-ctx.Done():
			return nil, trace.Wrap(err, "SSH timed out dialing %s", node.Addr())
		case <-time.After(retrySSH):
		}
	}
}

// Logf logs simultaneously to stdout and testing interface
func (g *gravity) Logf(format string, args ...interface{}) {
	elapsed := time.Since(g.ts)
	log.Printf("%s [%v %v] %s", g.param.Tag(), g, elapsed, fmt.Sprintf(format, args...))
	g.logFn("[%v %v] %s", g, elapsed, fmt.Sprintf(format, args...))
}

// String returns public and private addresses of the node
func (g *gravity) String() string {
	return fmt.Sprintf("%s %s", g.node.PrivateAddr(), g.node.Addr())
}

func (g *gravity) Node() infra.Node {
	return g.node
}

// Client returns SSH client to the node
func (g *gravity) Client() *ssh.Client {
	return g.ssh
}

// Install runs gravity install with params
func (g *gravity) Install(ctx context.Context, param InstallCmd) error {
	cmd := fmt.Sprintf("cd %s && sudo ./gravity install --debug --advertise-addr=%s --token=%s --flavor=%s --docker-device=%s --storage-driver=%s",
		g.param.installDir, g.node.PrivateAddr(), param.Token, param.Flavor, g.param.dockerDevice, g.param.storageDriver)

	err := sshutils.Run(ctx, g, cmd, nil)
	return trace.Wrap(err, cmd)
}

// SiteReport queries site report
// TODO: parse
func (g *gravity) SiteReport(ctx context.Context) error {
	return trace.Wrap(sshutils.Run(ctx, g, "gravity site report", nil))
}

// Status queries cluster status
func (g *gravity) Status(ctx context.Context) (*GravityStatus, error) {
	cmd := fmt.Sprintf("cd %s && sudo ./gravity status", g.param.installDir)
	status, exit, err := sshutils.RunAndParse(ctx, g, cmd, nil, parseStatus)

	if err != nil {
		return nil, trace.Wrap(err, cmd)
	}

	if exit != 0 {
		return nil, trace.Errorf("[%s/%s] %s returned %d",
			g.Node().PrivateAddr(), g.Node().Addr(), cmd, exit)
	}

	return status.(*GravityStatus), nil
}

func (g *gravity) OfflineUpdate(ctx context.Context, installerUrl string) error {
	return nil
}

// autoVals are set by command itself based on configuration
type autoVals struct{ InstallDir, PrivateAddr, DockerDevice string }

// cmdEx are extended parameters passed to gravity
type cmdEx struct {
	P   autoVals
	Cmd interface{}
}

var joinCmdTemplate = template.Must(
	template.New("gravity_join").Parse(
		`cd {{.P.InstallDir}} && sudo ./gravity join {{.Cmd.PeerAddr}} \
		--advertise-addr={{.P.PrivateAddr}} --token={{.Cmd.Token}} --debug \
		--role={{.Cmd.Role}} --docker-device={{.P.DockerDevice}}`))

func (g *gravity) Join(ctx context.Context, cmd JoinCmd) error {
	var buf bytes.Buffer
	err := joinCmdTemplate.Execute(&buf, cmdEx{
		P:   autoVals{g.param.installDir, g.Node().PrivateAddr(), g.param.dockerDevice},
		Cmd: cmd,
	})
	if err != nil {
		return trace.Wrap(err, buf.String())
	}

	err = sshutils.Run(ctx, g, buf.String(), nil)
	return trace.Wrap(err, cmd)
}

// Leave makes given node leave the cluster
func (g *gravity) Leave(ctx context.Context, graceful bool) error {
	var cmd string
	if graceful {
		cmd = fmt.Sprintf(`cd %s && sudo ./gravity leave --debug --confirm`, g.param.installDir)
	} else {
		cmd = fmt.Sprintf(`cd %s && sudo ./gravity leave --debug --confirm --force`, g.param.installDir)
	}

	err := sshutils.Run(ctx, g, cmd, nil)
	return trace.Wrap(err, cmd)
}

// Remove ejects node from cluster
func (g *gravity) Remove(ctx context.Context, node string, graceful bool) error {
	var cmd string
	if graceful {
		cmd = fmt.Sprintf(`cd %s && sudo ./gravity remove --debug --confirm %s`, g.param.installDir, node)
	} else {
		cmd = fmt.Sprintf(`cd %s && sudo ./gravity remove --debug --confirm --force %s`, g.param.installDir, node)
	}
	err := sshutils.Run(ctx, g, cmd, nil)
	return trace.Wrap(err, cmd)
}

// Uninstall removes gravity installation. It requires Leave beforehand
func (g *gravity) Uninstall(ctx context.Context) error {
	cmd := fmt.Sprintf(`cd %s && sudo ./gravity system uninstall --confirm`, g.param.installDir)
	err := sshutils.Run(ctx, g, cmd, nil)
	return trace.Wrap(err, cmd)
}

// PowerOff forcibly halts a machine
func (g *gravity) PowerOff(ctx context.Context, graceful bool) error {
	var cmd string
	if graceful {
		cmd = "sudo shutdown -h now"
	} else {
		cmd = "sudo poweroff -f"
	}

	sshutils.RunAndParse(ctx, g, cmd, nil, nil)
	g.ssh = nil
	// TODO: reliably destinguish between force close of SSH control channel and command being unable to run
	return nil
}

// Reboot gracefully restarts a machine and waits for it to become available again
func (g *gravity) Reboot(ctx context.Context, graceful bool) error {
	var cmd string
	if graceful {
		cmd = "sudo shutdown -r now"
	} else {
		cmd = "sudo reboot -f"
	}
	sshutils.RunAndParse(ctx, g, cmd, nil, nil)
	// TODO: reliably destinguish between force close of SSH control channel and command being unable to run

	client, err := sshClient(ctx, g.logFn, g.Node())
	if err != nil {
		return trace.Wrap(err, "SSH reconnect")
	}

	g.ssh = client
	return nil
}

// PullLogs fetches essential logs from the host and stores them in state dir
func (g *gravity) CollectLogs(ctx context.Context, prefix string) (string, error) {
	if g.ssh == nil {
		return "", trace.AccessDenied("node %v is poweroff", g)
	}

	files := []string{
		"/var/lib/gravity/planet/log",
		fmt.Sprintf("%s/*.log", g.param.installDir),
	}
	localPath := filepath.Join(g.param.stateDir, "node-logs", prefix, fmt.Sprintf("%s-logs.tgz", g.Node().PrivateAddr()))
	return localPath, trace.Wrap(sshutils.GetTgz(ctx, g, files, localPath))
}
