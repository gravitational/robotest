package gravity

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/terraform"
	"github.com/gravitational/robotest/lib/constants"
	sshutil "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"

	"github.com/dustin/go-humanize"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// cloudDynamicParams is a necessary evil to marry terraform vars, e2e legacy objects and needs of this provisioner
type cloudDynamicParams struct {
	ProvisionerConfig
	user    string
	homeDir string
	tf      terraform.Config
	env     map[string]string

	// options
	syncClocks bool
	waitDisks  bool
}

func configureVMs(baseCtx context.Context, log logrus.FieldLogger, params cloudDynamicParams, nodes []infra.Node) ([]Gravity, error) {
	errChan := make(chan error, len(nodes))
	nodeChan := make(chan interface{}, len(nodes))

	ctx, cancel := context.WithCancel(baseCtx)
	defer cancel()

	for _, node := range nodes {
		go func(node infra.Node) {
			val, err := configureVM(ctx, log, node, params)
			nodeChan <- val
			errChan <- err
		}(node)
	}

	nodeVals, err := utils.Collect(ctx, cancel, errChan, nodeChan)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	gravityNodes := []Gravity{}
	for _, node := range nodeVals {
		gravityNodes = append(gravityNodes, node.(Gravity))
	}

	return sorted(gravityNodes), nil
}

// CheckpointExists will check if checkpoint with given parameters exists, returns nil, error otherwise
func (c *TestContext) CheckpointExists(cloudProvider, checkpoint string, param interface{}) error {
	if c.suite.imageRegistry == nil {
		return trace.BadParameter("no image registry")
	}

	_, err := c.suite.imageRegistry.Locate(c.Context(),
		cloudProvider, checkpoint, param)
	return trace.Wrap(err)
}

// AssertNoSuchCheckpoint will gracefully abort test in case we're in snapshot capture mode and such checkpoint already exists
func (c *TestContext) AssertNoSuchCheckpoint(cloudProvider, checkpoint string, param interface{}) {
	if c.suite.imageRegistry == nil || c.suite.vmCaptureEnabled == false {
		return
	}

	img, err := c.suite.imageRegistry.Locate(c.Context(),
		cloudProvider, checkpoint, param)
	if err == nil {
		// avoid duplicates
		c.Logger().Infof("VM image already exists: %+v", img)
		c.checkpointSaved = true
		panic("VM image already exists")
	}

	if trace.IsNotFound(err) {
		// proceed with creating image
		return
	}

	c.OK("unexpected image registry issue", trace.Wrap(err))
}

func getSingleInstallDir(nodes []Gravity) (dir string, err error) {
	for _, node := range nodes {
		gn, ok := node.(*gravity)
		if !ok {
			return "", trace.BadParameter("invalid type")
		}

		if dir == "" {
			gn.installDir = dir
		}

		if dir != gn.installDir {
			return "", trace.BadParameter("all nodes must have uniform current installer dir")
		}
	}

	return dir, nil
}

// Checkpoint will save test state as VM image
func (c *TestContext) Checkpoint(checkpoint string, nodes []Gravity) {
	if !c.suite.vmCaptureEnabled {
		return
	}

	if c.vmCapture == nil || c.suite.imageRegistry == nil {
		c.Logger().Warn("Cannot make snapshot: VM capture/registry not available")
		return
	}

	installDir, err := getSingleInstallDir(nodes)
	if err != nil {
		c.Logger().WithError(err).Error("install dir across nodes")
		return
	}

	c.Logger().Infof("making checkpoint %q, param=%+v", checkpoint, c.param)

	err = c.deprovision(nodes)
	c.OK("deprovision", err)

	image, err := c.vmCapture.CaptureVM(c.Context())
	image.InstallDir = installDir

	c.OK("VM checkpoint capture", trace.Wrap(err))

	err = c.suite.imageRegistry.Store(c.Context(), checkpoint, c.param, *image)
	c.OK("save VM image info to registry", trace.Wrap(err))

	c.checkpointSaved = true
	panic(checkpoint)
}

// RestoreCheckpoint will attempt to recover VMs with compatible configuration at given checkpoint
func (c *TestContext) RestoreCheckpoint(cfg ProvisionerConfig, checkpoint string, param interface{}) (nodes []Gravity, err error) {
	if c.suite.imageRegistry == nil {
		return nil, trace.NotFound("image registry service unavailable")
	}

	cfg.FromImage, err = c.suite.imageRegistry.Locate(c.Context(),
		cfg.CloudProvider, checkpoint, param)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	c.Logger().WithField("checkpoint", checkpoint).Infof("using checkpoint %q images from %+v", checkpoint, cfg.FromImage)

	nodes, err = c.Provision(cfg)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	for _, n := range nodes {
		n.(*gravity).installDir = cfg.FromImage.InstallDir
	}
	return nodes, nil
}

// Provision gets VMs up, running and ready to use
func (c *TestContext) Provision(cfg ProvisionerConfig) ([]Gravity, error) {
	// store the configuration used for provisioning
	c.provisionerCfg = cfg

	var nodes []Gravity
	var destroy DestroyFn
	var err error
	switch cfg.CloudProvider {
	case constants.Azure, constants.AWS:
		nodes, destroy, err = c.provisionCloud(cfg)
	case constants.Ops:
		nodes, destroy, err = c.provisionOps(cfg)
	default:
		err = trace.BadParameter("unknown cloud provider: %v", cfg.CloudProvider)
	}

	if err == nil {
		c.resourceDestroyFuncs = append(c.resourceDestroyFuncs, destroy)
		return nodes, nil
	}

	if destroy == nil {
		return nil, trace.Wrap(err)
	}

	return nil, trace.NewAggregate(err, destroy())
}

// provisionOps utilizes an ops center installation flow to complete cluster installation
func (c *TestContext) provisionOps(cfg ProvisionerConfig) ([]Gravity, DestroyFn, error) {
	c.Logger().WithField("config", cfg).Debug("Provisioning via Ops Center")

	// verify connection before starting provisioning
	c.Logger().Debug("attempting to connect to AWS api")
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(cfg.Ops.EC2Region),
		Credentials: credentials.NewStaticCredentials(cfg.Ops.EC2AccessKey, cfg.Ops.EC2SecretKey, ""),
	})
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	c.Logger().Debug("logging into the ops center")
	out, err := exec.Command("tele", "login", "-o", cfg.Ops.URL, "--key", cfg.Ops.OpsKey).CombinedOutput()
	if err != nil {
		return nil, nil, trace.WrapWithMessage(err, string(out))
	}

	// generate a random cluster name
	clusterName := fmt.Sprint(c.name, "-", uuid.NewV4().String())
	c.provisionerCfg.clusterName = clusterName
	c.Logger().Debug("Generated cluster name: ", clusterName)

	c.Logger().Debug("validating configuration")
	err = validateConfig(cfg)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	c.Logger().Debug("generating ops center cluster configuration")
	clusterPath := path.Join(cfg.StateDir, "cluster.yaml")
	err = os.MkdirAll(cfg.StateDir, constants.SharedDirMask)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	defn, err := generateClusterConfig(cfg, clusterName)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	err = ioutil.WriteFile(clusterPath, []byte(defn), constants.SharedReadMask)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	// next, we need to tell the ops center to create our cluster
	c.Logger().Debug("requesting ops center to provision our cluster")
	out, err = exec.Command("tele", "create", clusterPath).CombinedOutput()
	if err != nil {
		return nil, nil, trace.Wrap(err, string(out))
	}

	// destroyFn defines the clean up function that will destroy provisioned resources
	destroyFn := cfg.DestroyOpsFn(c, clusterName)

	// monitor the cluster until it's created or times out
	timeout := time.After(cloudInitTimeout)
	tick := time.Tick(15 * time.Second)

	c.Logger().Debug("waiting for ops center provisioning to complete")
Loop:
	for {
		select {
		case <-timeout:
			return nil, destroyFn, errors.New("clusterInitTimeout exceeded")
		case <-tick:
			// check provisioning status
			status, err := getTeleClusterStatus(clusterName)
			c.Logger().WithField("status", status).Debug("provisioning status")
			if err != nil {
				return nil, destroyFn, trace.Wrap(err)
			}

			switch status {
			case "installing":
				// we're still installing, just continue the loop
			case "active":
				// the cluster install completed, we can continue the install process
				break Loop
			default:
				return nil, destroyFn, trace.BadParameter("unexpected cluster status: %v", status)
			}
		}
	}

	// now that the requested cluster has been created, we have to build the []Gravity slice of nodes
	c.Logger().Debug("attempting to get a listing of instances from AWS for the cluster")
	ec2svc := ec2.New(sess)
	gravityNodes, err := c.getAWSNodes(ec2svc, "tag:KubernetesCluster", clusterName)
	if err != nil {
		return nil, destroyFn, trace.Wrap(err)
	}

	ctx, cancel := context.WithTimeout(c.Context(), cloudInitTimeout)
	defer cancel()

	c.Logger().Debugf("running post provisioning tasks")
	err = c.postProvision(cfg, gravityNodes)
	if err != nil {
		return nil, destroyFn, trace.Wrap(err)
	}

	c.Logger().Debug("ensuring disk speed is adequate across nodes")
	ctx, cancel = context.WithTimeout(c.Context(), diskWaitTimeout)
	defer cancel()
	err = waitDisks(ctx, gravityNodes, []string{"/iotest", path.Join(cfg.dockerDevice, "/iotest")})
	if err != nil {
		err = trace.Wrap(err, "VM disks do not meet minimum write performance requirements")
		c.Logger().WithError(err).Error(err.Error())
		return nil, destroyFn, trace.Wrap(err)
	}

	c.Logger().Info("provisioning complete")
	return gravityNodes, destroyFn, nil
}

// provisionCloud gets VMs up, running and ready to use
func (c *TestContext) provisionCloud(cfg ProvisionerConfig) ([]Gravity, DestroyFn, error) {
	c.Logger().WithField("config", cfg).Debug("Provisioning VMs")

	err := validateConfig(cfg)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	nodes, destroyFn, vmCapture, params, err := runTerraform(c.Context(), cfg, c.Logger())
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	if c.suite.vmCaptureEnabled {
		c.vmCapture = vmCapture
	}

	ctx, cancel := context.WithTimeout(c.Context(), cloudInitTimeout)
	defer cancel()

	c.Logger().Debug("Configuring VMs")
	gravityNodes, err := configureVMs(ctx, c.Logger(), *params, nodes)
	if err != nil {
		c.Logger().WithError(err).Error("some nodes initialization failed, teardown this setup as non-usable")
		return nil, nil, trace.NewAggregate(err, destroyFn(ctx))
	}

	err = c.postProvision(cfg, gravityNodes)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	c.Logger().Debug("ensuring disk speed is adequate across nodes")
	ctx, cancel = context.WithTimeout(c.Context(), diskWaitTimeout)
	defer cancel()
	err = waitDisks(ctx, gravityNodes, []string{"/iotest", cfg.dockerDevice})
	if err != nil {
		err = trace.Wrap(err, "VM disks do not meet minimum write performance requirements")
		c.Logger().WithError(err).Error(err.Error())
		return nil, nil, trace.NewAggregate(err, destroyFn(ctx))
	}

	return gravityNodes, wrapDestroyFn(c, cfg.Tag(), gravityNodes, destroyFn), nil
}

// postProvision runs common tasks for both ops and cloud provisioners once the VMs have been setup and are running
func (c *TestContext) postProvision(cfg ProvisionerConfig, gravityNodes []Gravity) error {
	c.Logger().Debug("streaming logs")
	for _, node := range gravityNodes {
		go node.(*gravity).streamLogs(c.Context())
	}

	ctx, cancel := context.WithTimeout(c.Context(), clockSyncTimeout)
	defer cancel()

	c.Logger().Debug("synchronizing clocks")
	timeNodes := []sshutil.SshNode{}
	for _, node := range gravityNodes {
		timeNodes = append(timeNodes, sshutil.SshNode{node.Client(), node.Logger()})
	}
	if err := sshutil.WaitTimeSync(ctx, timeNodes); err != nil {
		return trace.Wrap(err)
	}
	return nil
}

// sort Interface implementation
type byPrivateAddr []Gravity

func (g byPrivateAddr) Len() int      { return len(g) }
func (g byPrivateAddr) Swap(i, j int) { g[i], g[j] = g[j], g[i] }
func (g byPrivateAddr) Less(i, j int) bool {
	return g[i].Node().PrivateAddr() < g[j].Node().PrivateAddr()
}

func sorted(nodes []Gravity) []Gravity {
	sort.Sort(byPrivateAddr(nodes))
	return nodes
}

const (
	// set by https://github.com/Azure/WALinuxAgent which is bundled with all Azure linux distro
	// once basic provisioning procedures are complete
	waagentProvisionFile   = "/var/lib/waagent/provisioned"
	cloudInitSupportedFile = "/var/lib/bootstrap_started"
	cloudInitCompleteFile  = "/var/lib/bootstrap_complete"
)

const cloudInitWait = time.Second * 10

// bootstrapAzure workarounds some issues with Azure platform init
func bootstrapAzure(ctx context.Context, g Gravity, param cloudDynamicParams) (err error) {
	err = sshutil.WaitForFile(ctx, g.Client(), g.Logger(),
		waagentProvisionFile, sshutil.TestRegularFile)
	if err != nil {
		return trace.Wrap(err)
	}

	err = sshutil.TestFile(ctx, g.Client(), g.Logger(), cloudInitCompleteFile, sshutil.TestRegularFile)
	if err == nil {
		g.Logger().Debug("node already bootstrapped")
		return nil
	}
	if !trace.IsNotFound(err) {
		return trace.Wrap(err)
	}

	err = sshutil.TestFile(ctx, g.Client(), g.Logger(), cloudInitSupportedFile, sshutil.TestRegularFile)
	if err == nil {
		g.Logger().Debug("cloud-init underway")
		return sshutil.WaitForFile(ctx, g.Client(), g.Logger(), cloudInitCompleteFile, sshutil.TestRegularFile)
	}
	if !trace.IsNotFound(err) {
		return trace.Wrap(err)
	}

	// apparently cloud-init scripts are not supported for given OS
	err = sshutil.RunScript(ctx, g.Client(), g.Logger(),
		filepath.Join(param.ScriptPath, "bootstrap", fmt.Sprintf("%s.sh", param.os.Vendor)),
		sshutil.SUDO)
	return trace.Wrap(err)
}

// bootstrapAWS is a simple workflow to wait for cloud-init to complete
func bootstrapAWS(ctx context.Context, g Gravity, param cloudDynamicParams) (err error) {
	err = sshutil.WaitForFile(ctx, g.Client(), g.Logger(), cloudInitCompleteFile, sshutil.TestRegularFile)
	if err != nil {
		return trace.Wrap(err)
	}

	return trace.Wrap(err)
}

// ConfigureNode is used to configure a provisioned node
// 1. wait for node to boot
// 2. (TODO) run bootstrap scripts - as Azure doesn't support them for RHEL/CentOS, will migrate here
// 2.  - i.e. run bootstrap commands, load installer, etc.
// TODO: migrate bootstrap scripts here as well;
func configureVM(ctx context.Context, log logrus.FieldLogger, node infra.Node, param cloudDynamicParams) (Gravity, error) {
	g := &gravity{
		node:  node,
		param: param,
		ts:    time.Now(),
		log: log.WithFields(logrus.Fields{
			"ip":        node.PrivateAddr(),
			"public_ip": node.Addr(),
		}),
	}

	client, err := sshClient(ctx, g.node, g.log)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	g.ssh = client

	switch param.CloudProvider {
	case constants.AWS:
		err = bootstrapAWS(ctx, g, param)
	case constants.Azure:
		err = bootstrapAzure(ctx, g, param)
	case "ops":
		// For ops installs, we don't run the installer. So just hardcode the install directory to /bin
		g.installDir = "/bin"
	default:
		return nil, trace.BadParameter("unsupported cloud provider %s", param.CloudProvider)
	}
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return g, nil
}

// waitDisks is a necessary workaround for Azure VMs to wait until their disk initialization processes are complete
// otherwise it'll fail telekube pre-install checks
func waitDisks(ctx context.Context, nodes []Gravity, paths []string) error {
	errs := make(chan error, len(nodes))

	for _, node := range nodes {
		go func(node Gravity) {
			errs <- waitDisk(ctx, node, paths, minDiskSpeed)
		}(node)
	}

	return trace.Wrap(utils.CollectErrors(ctx, errs))
}

// waitDisk will wait specific disk performance to report OK
func waitDisk(ctx context.Context, node Gravity, paths []string, minSpeed uint64) error {
	err := wait.Retry(ctx, func() error {
		for _, p := range paths {
			if !strings.HasPrefix(p, "/dev") {
				defer sshutil.Run(ctx, node.Client(), node.Logger(), fmt.Sprintf("sudo /bin/rm -f %s", p), nil)
			}
			var out string
			_, err := sshutil.RunAndParse(ctx, node.Client(), node.Logger(),
				fmt.Sprintf("sudo dd if=/dev/zero of=%s bs=100K count=1024 conv=fdatasync 2>&1", p),
				nil, sshutil.ParseAsString(&out))
			if err != nil {
				return wait.Abort(trace.Wrap(err))
			}
			speed, err := ParseDDOutput(out)
			if err != nil {
				return wait.Abort(trace.Wrap(err))
			}
			if speed < minSpeed {
				return wait.Continue(fmt.Sprintf("%s has %v/s < minimum of %v/s",
					p, humanize.Bytes(speed), humanize.Bytes(minSpeed)))
			}
		}
		return nil
	})
	return trace.Wrap(err)
}
