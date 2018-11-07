package gravity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/providers/gce"
	"github.com/gravitational/robotest/lib/constants"
	"github.com/gravitational/robotest/lib/defaults"
	sshutils "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/wait"
	"github.com/gravitational/trace"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// Gravity is interface to remote gravity CLI
type Gravity interface {
	json.Marshaler
	// SetInstaller transfers and prepares installer package
	SetInstaller(ctx context.Context, installerUrl string, subdir string) error
	// ExecScript transfers and executes script with predefined parameters
	ExecScript(ctx context.Context, scriptUrl string, args []string) error
	// Install operates on initial master node
	Install(ctx context.Context, param InstallParam) error
	// Status retrieves status
	Status(ctx context.Context) (*GravityStatus, error)
	// OfflineUpdate tries to upgrade application version
	OfflineUpdate(ctx context.Context, installerUrl string) error
	// Join asks to join existing cluster (or installation in progress)
	Join(ctx context.Context, param JoinCmd) error
	// Leave requests current node leave a cluster
	Leave(ctx context.Context, graceful Graceful) error
	// Remove requests cluster to evict a given node
	Remove(ctx context.Context, node string, graceful Graceful) error
	// Uninstall will wipe gravity installation from node
	Uninstall(ctx context.Context) error
	// PowerOff will power off the node
	PowerOff(ctx context.Context, graceful Graceful) error
	// Reboot will reboot this node and wait until it will become available again
	Reboot(ctx context.Context, graceful Graceful) error
	// CollectLogs will pull essential logs from node and store it in state dir under node-logs/prefix
	CollectLogs(ctx context.Context, prefix string) (localPath string, err error)
	// Upload uploads packages in current installer dir to cluster
	Upload(ctx context.Context) error
	// Upgrade takes currently active installer (see SetInstaller) and tries to perform upgrade
	Upgrade(ctx context.Context) error
	// RunInPlanet runs specific command inside Planet container and returns its result
	RunInPlanet(ctx context.Context, cmd string, args ...string) (string, error)
	// Node returns underlying VM instance
	Node() infra.Node
	// Offline returns true if node was previously powered off
	Offline() bool
	// Client returns SSH client to VM instance
	Client() *ssh.Client
	// Text representation
	String() string
	// Will log using extended info such as current tag, node info, etc
	Logger() logrus.FieldLogger
}

type Graceful bool

// InstallParam represents install parameters passed to first node
type InstallParam struct {
	// Token is initial token to use during cluster setup
	Token string `json:"-"`
	// Role is node role as defined in app.yaml
	Role string `json:"role" validate:"required"`
	// Cluster is Optional name of the cluster. Autogenerated if not set.
	Cluster string `json:"cluster"`
	// Flavor is Application flavor. See Application Manifest for details.
	Flavor string `json:"flavor" validate:"required"`
	// K8SConfigURL is (Optional) File with Kubernetes resources to create in the cluster during installation.
	K8SConfigURL string `json:"k8s_config_url,omitempty"`
	// PodNetworkCidr is (Optional) CIDR range Kubernetes will be allocating node subnets and pod IPs from. Must be a minimum of /16 so Kubernetes is able to allocate /24 to each node. Defaults to 10.244.0.0/16.
	PodNetworkCIDR string `json:"pod_network_cidr,omitempty"`
	// ServiceCidr (Optional) CIDR range Kubernetes will be allocating service IPs from. Defaults to 10.100.0.0/16.
	ServiceCIDR string `json:"service_cidr,omitempty"`
	// EnableRemoteSupport (Optional) whether to register this installation with remote ops-center
	EnableRemoteSupport bool `json:"remote_support"`
	// LicenseURL (Optional) is license file, could be local or s3 or http(s) url
	LicenseURL string `json:"license,omitempty"`
	// CloudProvider defines tighter integration with cloud vendor, i.e. use AWS networking on Amazon
	CloudProvider string `json:"cloud_provider,omitempty"`
	// GCENodeTag specifies the node tag on GCE.
	// Node tag replaces the cluster name if the cluster name does not comply with the GCE naming convention
	GCENodeTag string `json:"gce_node_tag"`
	// StateDir is the directory where all gravity data will be stored on the node
	StateDir string `json:"state_dir" validate:"required"`
	// OSFlavor is operating system and optional version separated by ':'
	OSFlavor OS `json:"os" validate:"required"`
	// DockerStorageDriver is one of supported storage drivers
	DockerStorageDriver StorageDriver `json:"storage_driver"`
	// InstallerURL overrides installer URL from the global config
	InstallerURL string `json:"installer_url,omitempty"`
	// OpsAdvertiseAddr is optional Ops Center advertise address to pass to the install command
	OpsAdvertiseAddr string `json:"ops_advertise_addr,omitempty"`
}

// JoinCmd represents various parameters for Join
type JoinCmd struct {
	// InstallDir is set automatically
	InstallDir string
	// PeerAddr is other node (i.e. master)
	PeerAddr string
	// Token is the join token
	Token string
	// Role is the role of the joining node
	Role string
	// StateDir is where all gravity data will be stored on the joining node
	StateDir string
}

// GravityStatus is serialized form of `gravity status` CLI.
type GravityStatus struct {
	Application string
	Cluster     string
	Status      string
	// Token is secure token which prevents rogue nodes from joining the cluster during installation
	Token string `validation:"required"`
	// Nodes defines nodes the cluster observes
	Nodes []string
}

type gravity struct {
	node       infra.Node
	ssh        *ssh.Client
	installDir string
	param      cloudDynamicParams
	ts         time.Time
	log        logrus.FieldLogger
}

func (g *gravity) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"public_ip": g.node.Addr(),
		"ip":        g.node.PrivateAddr(),
	})
}

// waits for SSH to be up on node and returns client
func sshClient(baseContext context.Context, node infra.Node, log logrus.FieldLogger) (*ssh.Client, error) {
	ctx, cancel := context.WithTimeout(baseContext, deadlineSSH)
	defer cancel()

	for {
		client, err := node.Client()

		if err == nil {
			log.Debug("connected via SSH")
			return client, nil
		}

		log.WithFields(logrus.Fields{"error": err, "retry_in": retrySSH}).Debug("waiting for SSH")
		select {
		case <-ctx.Done():
			log.WithError(ctx.Err()).Debug("context cancelled or timed out, SSH connection cancelled")
			return nil, trace.Wrap(err, "SSH timed out dialing %s", node.Addr())
		case <-time.After(retrySSH):
		}
	}
}

func (g *gravity) Logger() logrus.FieldLogger {
	return g.log
}

// String returns public and private addresses of the node
func (g *gravity) String() string {
	return fmt.Sprintf("%s/%s", g.node.PrivateAddr(), g.node.Addr())
}

func (g *gravity) Node() infra.Node {
	return g.node
}

// Client returns SSH client to the node
func (g *gravity) Client() *ssh.Client {
	return g.ssh
}

// Install runs gravity install with params
func (g *gravity) Install(ctx context.Context, param InstallParam) error {
	// cmd specify additional configuration for the install command
	// collected from defaults and/or computed values
	type cmd struct {
		InstallDir      string
		PrivateAddr     string
		EnvDockerDevice string
		StorageDriver   string
		InstallParam
	}

	config := cmd{
		InstallDir:      g.installDir,
		PrivateAddr:     g.Node().PrivateAddr(),
		EnvDockerDevice: constants.EnvDockerDevice,
		StorageDriver:   g.param.storageDriver.Driver(),
		InstallParam:    param,
	}
	if param.CloudProvider == constants.GCE {
		config.InstallParam.GCENodeTag = gce.TranslateClusterName(param.Cluster)
	}

	var buf bytes.Buffer
	err := installCmdTemplate.Execute(&buf, config)
	if err != nil {
		return trace.Wrap(err, buf.String())
	}

	err = sshutils.Run(ctx, g.Client(), g.Logger(), buf.String(), map[string]string{
		constants.EnvDockerDevice: g.param.dockerDevice,
	})
	return trace.Wrap(err, param)
}

var installCmdTemplate = template.Must(
	template.New("gravity_install").Parse(`
		source /tmp/gravity_environment >/dev/null 2>&1 || true; \
		cd {{.InstallDir}} && ./gravity version && sudo ./gravity install --debug \
		--advertise-addr={{.PrivateAddr}} --token={{.Token}} --flavor={{.Flavor}} \
		--docker-device=${{.EnvDockerDevice}} \
		{{if .StorageDriver}}--storage-driver={{.StorageDriver}}{{end}} \
		--system-log-file=./telekube-system.log \
		--cloud-provider={{.CloudProvider}} --state-dir={{.StateDir}} \
		{{if .Cluster}}--cluster={{.Cluster}}{{end}} \
		{{if .GCENodeTag}}--gce-node-tag={{.GCENodeTag}}{{end}} \
		{{if .OpsAdvertiseAddr}}--ops-advertise-addr={{.OpsAdvertiseAddr}}{{end}}
`))

// Status queries cluster status
func (g *gravity) Status(ctx context.Context) (*GravityStatus, error) {
	cmd := "sudo gravity status --system-log-file=./telekube-system.log"
	status := GravityStatus{}
	exit, err := sshutils.RunAndParse(ctx, g.Client(), g.Logger(), cmd, nil, parseStatus(&status))

	if err != nil {
		return nil, trace.Wrap(err, cmd)
	}

	if exit != 0 {
		return nil, trace.Errorf("[%s/%s] %s returned %d",
			g.Node().PrivateAddr(), g.Node().Addr(), cmd, exit)
	}

	return &status, nil
}

func (g *gravity) OfflineUpdate(ctx context.Context, installerUrl string) error {
	return nil
}

func (g *gravity) Join(ctx context.Context, param JoinCmd) error {
	// cmd specify additional configuration for the join command
	// collected from defaults and/or computed values
	type cmd struct {
		InstallDir, PrivateAddr, EnvDockerDevice string
		JoinCmd
	}

	var buf bytes.Buffer
	err := joinCmdTemplate.Execute(&buf, cmd{
		InstallDir:      g.installDir,
		PrivateAddr:     g.Node().PrivateAddr(),
		EnvDockerDevice: constants.EnvDockerDevice,
		JoinCmd:         param,
	})
	if err != nil {
		return trace.Wrap(err, buf.String())
	}

	err = sshutils.Run(ctx, g.Client(), g.Logger(), buf.String(), map[string]string{
		constants.EnvDockerDevice: g.param.dockerDevice,
	})
	return trace.Wrap(err, param)
}

var joinCmdTemplate = template.Must(
	template.New("gravity_join").Parse(`
		source /tmp/gravity_environment >/dev/null 2>&1 || true; \
		cd {{.InstallDir}} && sudo ./gravity join {{.PeerAddr}} \
		--advertise-addr={{.PrivateAddr}} --token={{.Token}} --debug \
		--role={{.Role}} --docker-device=${{.EnvDockerDevice}} \
		--system-log-file=./telekube-system.log --state-dir={{.StateDir}}`))

// Leave makes given node leave the cluster
func (g *gravity) Leave(ctx context.Context, graceful Graceful) error {
	var cmd string
	if graceful {
		cmd = `leave --confirm`
	} else {
		cmd = `leave --confirm --force`
	}

	return trace.Wrap(g.runOp(ctx, cmd))
}

// Remove ejects node from cluster
func (g *gravity) Remove(ctx context.Context, node string, graceful Graceful) error {
	var cmd string
	if graceful {
		cmd = fmt.Sprintf(`remove --confirm %s`, node)
	} else {
		cmd = fmt.Sprintf(`remove --confirm --force %s`, node)
	}
	return trace.Wrap(g.runOp(ctx, cmd))
}

// Uninstall removes gravity installation. It requires Leave beforehand
func (g *gravity) Uninstall(ctx context.Context) error {
	cmd := fmt.Sprintf(`cd %s && sudo ./gravity system uninstall --confirm --system-log-file=./telekube-system.log`, g.installDir)
	err := sshutils.Run(ctx, g.Client(), g.Logger(), cmd, nil)
	return trace.Wrap(err, cmd)
}

// PowerOff forcibly halts a machine
func (g *gravity) PowerOff(ctx context.Context, graceful Graceful) error {
	var cmd string
	if graceful {
		cmd = "sudo shutdown -h now"
	} else {
		cmd = "sudo poweroff -f"
	}

	sshutils.RunAndParse(ctx, g.Client(), g.Logger(), cmd, nil, nil)
	g.ssh = nil
	// TODO: reliably destinguish between force close of SSH control channel and command being unable to run
	return nil
}

func (g *gravity) Offline() bool {
	return g.ssh == nil
}

// Reboot gracefully restarts a machine and waits for it to become available again
func (g *gravity) Reboot(ctx context.Context, graceful Graceful) error {
	var cmd string
	if graceful {
		cmd = "sudo shutdown -r now"
	} else {
		cmd = "sudo reboot -f"
	}
	sshutils.RunAndParse(ctx, g.Client(), g.Logger(), cmd, nil, nil)
	// TODO: reliably destinguish between force close of SSH control channel and command being unable to run

	client, err := sshClient(ctx, g.Node(), g.Logger())
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

	localPath := filepath.Join(g.param.StateDir, "node-logs", prefix, fmt.Sprintf("%s-logs.tgz", g.Node().PrivateAddr()))
	return localPath, trace.Wrap(sshutils.PipeCommand(ctx, g.Client(), g.Logger(),
		fmt.Sprintf("cd %s && sudo ./gravity system report", g.installDir), localPath))
}

// SetInstaller overrides default installer into
func (g *gravity) SetInstaller(ctx context.Context, installerURL string, subdir string) error {
	installDir := filepath.Join(g.param.homeDir, subdir)
	log := g.Logger().WithFields(logrus.Fields{"installer_url": installerURL, "install_dir": installDir})

	log.Debugf("Set installer %v -> %v", installerURL, installDir)

	tgz, err := sshutils.TransferFile(ctx, g.Client(), log, installerURL, installDir, g.param.env)
	if err != nil {
		log.WithError(err).Error("Failed to transfer installer")
		return trace.Wrap(err)
	}

	err = sshutils.Run(ctx, g.Client(), log, fmt.Sprintf("tar -xvf %s -C %s", tgz, installDir), nil)
	if err != nil {
		return trace.Wrap(err)
	}

	g.installDir = installDir
	return nil
}

// ExecScript will transfer and execute script provided with given args
func (g *gravity) ExecScript(ctx context.Context, scriptUrl string, args []string) error {
	log := g.Logger().WithFields(logrus.Fields{
		"script": scriptUrl, "args": args})

	log.Debug("execute")

	spath, err := sshutils.TransferFile(ctx, g.Client(), log,
		scriptUrl, defaults.TmpDir, g.param.env)
	if err != nil {
		log.WithError(err).Error("failed to transfer script")
		return trace.Wrap(err)
	}

	err = sshutils.Run(ctx, g.Client(), log,
		fmt.Sprintf("sudo /bin/bash -x %s %s", spath, strings.Join(args, " ")), nil)
	return trace.Wrap(err)
}

// Upload uploads packages in current installer dir to cluster
func (g *gravity) Upload(ctx context.Context) error {
	err := sshutils.Run(ctx, g.Client(), g.Logger(), fmt.Sprintf(`cd %s && sudo ./upload`, g.installDir), nil)
	return trace.Wrap(err)
}

// Upgrade takes current installer and tries to perform upgrade
func (g *gravity) Upgrade(ctx context.Context) error {
	return trace.Wrap(g.runOp(ctx,
		fmt.Sprintf("upgrade $(./gravity app-package --state-dir=.) --etcd-retry-timeout=%v", defaults.EtcdRetryTimeout)))
}

// for cases when gravity doesn't return just opcode but an extended message
var reGravityExtended = regexp.MustCompile(`launched operation \"([a-z0-9\-]+)\".*`)

const (
	opStatusCompleted = "completed"
	opStatusFailed    = "failed"
)

// runOp launches specific command and waits for operation to complete, ignoring transient errors
func (g *gravity) runOp(ctx context.Context, command string) error {
	var code string
	_, err := sshutils.RunAndParse(ctx, g.Client(), g.Logger(),
		fmt.Sprintf(`cd %s && sudo ./gravity %s --insecure --quiet --system-log-file=./telekube-system.log`,
			g.installDir, command),
		nil, sshutils.ParseAsString(&code))
	if err != nil {
		return trace.Wrap(err)
	}
	if match := reGravityExtended.FindStringSubmatch(code); len(match) == 2 {
		code = match[1]
	}

	retry := wait.Retryer{
		Attempts:    1000,
		Delay:       time.Second * 20,
		FieldLogger: g.Logger().WithField("retry-operation", code),
	}

	err = retry.Do(ctx, func() error {
		var response string
		cmd := fmt.Sprintf(`cd %s && ./gravity status --operation-id=%s -q`, g.installDir, code)
		_, err := sshutils.RunAndParse(ctx, g.Client(), g.Logger(),
			cmd, nil, sshutils.ParseAsString(&response))
		if err != nil {
			return wait.Continue(cmd)
		}

		switch strings.TrimSpace(response) {
		case opStatusCompleted:
			return nil
		case opStatusFailed:
			return wait.Abort(trace.Errorf("%s: response=%s, err=%v", cmd, response, err))
		default:
			return wait.Continue("non-final / unknown op status: %q", response)
		}
	})
	return trace.Wrap(err)
}

// RunInPlanet executes given command inside Planet container
func (g *gravity) RunInPlanet(ctx context.Context, cmd string, args ...string) (string, error) {
	c := fmt.Sprintf(`cd %s && sudo ./gravity enter -- --notty %s -- %s`,
		g.installDir, cmd, strings.Join(args, " "))

	var out string
	_, err := sshutils.RunAndParse(ctx, g.Client(), g.Logger(), c, nil, sshutils.ParseAsString(&out))
	if err != nil {
		return "", trace.Wrap(err)
	}

	return out, nil
}
