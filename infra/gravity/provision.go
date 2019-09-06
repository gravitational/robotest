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

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/terraform"
	"github.com/gravitational/robotest/lib/constants"
	sshutil "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/robotest/lib/wait"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/dustin/go-humanize"
	"github.com/gravitational/trace"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// cloudDynamicParams is a necessary evil to marry terraform vars, e2e legacy objects and needs of this provisioner
type cloudDynamicParams struct {
	ProvisionerConfig
	user      string
	homeDir   string
	terraform terraform.Config
	env       map[string]string
}

func configureVMs(baseCtx context.Context, log logrus.FieldLogger, params cloudDynamicParams, nodes []*gravity) error {
	errChan := make(chan error, len(nodes))

	ctx, cancel := context.WithCancel(baseCtx)
	defer cancel()

	for _, node := range nodes {
		go func(node *gravity) {
			err := configureVM(ctx, log, node, params)
			errChan <- err
		}(node)
	}

	err := utils.CollectErrors(ctx, errChan)
	if err != nil {
		return trace.Wrap(err)
	}

	return nil
}

// Provision will attempt to provision the requested cluster
func (c *TestContext) Provision(cfg ProvisionerConfig) (cluster Cluster, err error) {
	// store the configuration used for provisioning
	c.provisionerCfg = cfg

	switch cfg.CloudProvider {
	case constants.Azure, constants.AWS, constants.GCE, constants.Libvirt:
		var config *terraform.Config
		cluster, config, err = c.provisionCloud(cfg)
		if err == nil && cfg.CloudProvider == constants.GCE {
			c.WithFields(logrus.Fields{"node_tag": config.GCE.NodeTag})
		}
	case constants.Ops:
		cluster, err = c.provisionOps(cfg)
	default:
		err = trace.BadParameter("unknown cloud provider: %q", cfg.CloudProvider)
	}

	// call `destroyFn` if provided to destroy infrastructure
	if err != nil && cluster.Destroy != nil {
		destroyErr := cluster.Destroy()
		if destroyErr != nil {
			err = trace.NewAggregate(err, destroyErr)
		}
		// if we destroy the cluster, make sure nodes and the destroy functions are returned as nil
		cluster.Nodes = nil
		cluster.Destroy = nil
	}

	return cluster, trace.Wrap(err)
}

// provisionOps utilizes an ops center installation flow to complete cluster installation
func (c *TestContext) provisionOps(cfg ProvisionerConfig) (cluster Cluster, err error) {
	c.Logger().WithField("config", cfg).Debug("Provisioning via Ops Center")

	// verify connection before starting provisioning
	c.Logger().Debug("attempting to connect to AWS api")
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(cfg.Ops.EC2Region),
		Credentials: credentials.NewStaticCredentials(cfg.Ops.EC2AccessKey, cfg.Ops.EC2SecretKey, ""),
	})
	if err != nil {
		return cluster, trace.Wrap(err)
	}
	c.Logger().Debug("logging into the ops center")
	out, err := exec.Command("tele", "login", "-o", cfg.Ops.URL, "--key", cfg.Ops.OpsKey).CombinedOutput()
	if err != nil {
		return cluster, trace.WrapWithMessage(err, string(out))
	}

	// generate a random cluster name
	clusterName := fmt.Sprint(c.name, "-", uuid.NewV4().String())
	c.provisionerCfg.clusterName = clusterName
	c.Logger().Debug("Generated cluster name: ", clusterName)

	c.Logger().Debug("validating configuration")
	err = validateConfig(cfg)
	if err != nil {
		return cluster, trace.Wrap(err)
	}

	c.Logger().Debug("generating ops center cluster configuration")
	clusterPath := path.Join(cfg.StateDir, "cluster.yaml")
	err = os.MkdirAll(cfg.StateDir, constants.SharedDirMask)
	if err != nil {
		return cluster, trace.Wrap(err)
	}
	defn, err := generateClusterConfig(cfg, clusterName)
	if err != nil {
		return cluster, trace.Wrap(err)
	}
	err = ioutil.WriteFile(clusterPath, []byte(defn), constants.SharedReadMask)
	if err != nil {
		return cluster, trace.Wrap(err)
	}

	// next, we need to tell the ops center to create our cluster
	c.Logger().Debug("requesting ops center to provision our cluster")
	out, err = exec.Command("tele", "create", clusterPath).CombinedOutput()
	if err != nil {
		return cluster, trace.Wrap(err, string(out))
	}

	// destroyFn defines the clean up function that will destroy provisioned resources
	cluster.Destroy = cfg.DestroyOpsFn(c, clusterName)

	// monitor the cluster until it's created or times out
	timeout := time.After(cloudInitTimeout)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	c.Logger().Debug("waiting for ops center provisioning to complete")
Loop:
	for {
		select {
		case <-timeout:
			return cluster, errors.New("clusterInitTimeout exceeded")
		case <-ticker.C:
			// check provisioning status
			status, err := getTeleClusterStatus(clusterName)
			c.Logger().WithField("status", status).Debug("provisioning status")
			if err != nil {
				return cluster, trace.Wrap(err)
			}

			switch status {
			case "installing":
				// we're still installing, just continue the loop
			case "active":
				// the cluster install completed, we can continue the install process
				break Loop
			default:
				return cluster, trace.BadParameter("unexpected cluster status: %v", status)
			}
		}
	}

	// now that the requested cluster has been created, we have to build the []Gravity slice of nodes
	c.Logger().Debug("Attempting to get a listing of instances from AWS for the cluster.")
	ec2svc := ec2.New(sess)
	gravityNodes, err := c.getAWSNodes(ec2svc, "tag:KubernetesCluster", clusterName)
	if err != nil {
		return cluster, trace.Wrap(err)
	}

	c.Logger().Debugf("Running post provisioning tasks.")
	err = c.postProvision(gravityNodes)
	if err != nil {
		return cluster, trace.Wrap(err)
	}

	err = validateDiskSpeed(c.Context(), gravityNodes, cfg.dockerDevice, c.Logger())
	if err != nil {
		return cluster, trace.Wrap(err)
	}

	cluster.Nodes = asNodes(gravityNodes)
	c.Logger().Info("Provisioning complete.")
	return cluster, nil
}

// provisionCloud gets VMs up, running and ready to use
func (c *TestContext) provisionCloud(cfg ProvisionerConfig) (cluster Cluster, config *terraform.Config, err error) {
	log := c.Logger().WithField("config", cfg)
	log.Debug("Provisioning VMs.")

	err = validateConfig(cfg)
	if err != nil {
		return cluster, nil, trace.Wrap(err)
	}

	infra, err := runTerraform(c.Context(), cfg, c.Logger())
	if err != nil {
		return cluster, nil, trace.Wrap(err)
	}
	defer func() {
		if err == nil || infra.destroyFn == nil {
			return
		}
		if errDestroy := destroyResource(infra.destroyFn); errDestroy != nil {
			log.WithError(errDestroy).Error("Failed to destroy resources.")
		}
	}()

	ctx, cancel := context.WithTimeout(c.Context(), cloudInitTimeout)
	defer cancel()

	log.Debug("Connecting to VMs.")
	gravityNodes, err := connectVMs(ctx, c.Logger(), infra.params, infra.nodes)
	if err != nil {
		log.WithError(err).Error("Some nodes failed to connect, tear down as unusable.")
		return cluster, nil, trace.NewAggregate(err, destroyResource(infra.destroyFn))
	}
	// Start streaming logs as soon as connected
	c.streamLogs(gravityNodes)

	log.Debug("Configuring VMs.")
	err = configureVMs(ctx, c.Logger(), infra.params, gravityNodes)
	if err != nil {
		log.WithError(err).Error("Some nodes failed to initialize, tear down as non-usable.")
		return cluster, nil, trace.NewAggregate(err, destroyResource(infra.destroyFn))
	}

	err = c.postProvision(gravityNodes)
	if err != nil {
		log.WithError(err).Error("Post-provisioning failed, tear down as non-usable.")
		return cluster, nil, trace.Wrap(err)
	}

	if cfg.CloudProvider == constants.Azure {
		err = validateDiskSpeed(c.Context(), gravityNodes, cfg.dockerDevice, c.Logger())
		if err != nil {
			return cluster, nil, trace.Wrap(err)
		}
	}

	log.WithField("nodes", gravityNodes).Debug("Provisioning complete.")

	nodes := asNodes(gravityNodes)
	cluster.Nodes = nodes
	cluster.Destroy = wrapDestroyFunc(c, cfg.Tag(), nodes, infra.destroyFn)

	return cluster, &infra.params.terraform, nil
}

func (c *TestContext) streamLogs(gravityNodes []*gravity) {
	c.Logger().Debug("Streaming logs.")
	for _, node := range gravityNodes {
		go func(node *gravity) {
			err := node.streamStartupLogs(c.monitorCtx)
			if err != nil && !utils.IsContextCancelledError(err) {
				c.Logger().Warnf("Failed to stream startup script logs: %v.", err)
			}
		}(node)
		go func(node *gravity) {
			if err := node.streamLogs(c.monitorCtx); err != nil {
				switch {
				case sshutil.IsExitMissingError(err):
					if c.Context().Err() != nil {
						// This test has already been cancelled / has timed out
						return
					}
					c.markPreempted(node)
				case utils.IsContextCancelledError(err):
					// Ignore
				default:
					c.Logger().Warnf("Failed to stream logs: %v.", err)
				}
			}
		}(node)
	}
}

// postProvision runs common tasks for both ops and cloud provisioners once the VMs have been setup and are running
func (c *TestContext) postProvision(gravityNodes []*gravity) error {
	ctx, cancel := context.WithTimeout(c.Context(), clockSyncTimeout)
	defer cancel()

	c.Logger().Debug("synchronizing clocks")
	var timeNodes []sshutil.SshNode
	for _, node := range gravityNodes {
		timeNodes = append(timeNodes, sshutil.SshNode{Client: node.Client(), Log: node.Logger()})
	}
	if err := sshutil.WaitTimeSync(ctx, timeNodes); err != nil {
		return trace.Wrap(err)
	}
	return nil
}

const (
	// set by https://github.com/Azure/WALinuxAgent which is bundled with all Azure linux distro
	// once basic provisioning procedures are complete
	waagentProvisionFile   = "/var/lib/waagent/provisioned"
	cloudInitSupportedFile = "/var/lib/bootstrap_started"
	cloudInitCompleteFile  = "/var/lib/bootstrap_complete"
)

// bootstrapAzure workarounds some issues with Azure platform init
func bootstrapAzure(ctx context.Context, g *gravity, param cloudDynamicParams) (err error) {
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

// bootstrapCloud is a simple workflow to wait for cloud-init to complete
func bootstrapCloud(ctx context.Context, g *gravity, param cloudDynamicParams) (err error) {
	err = sshutil.WaitForFile(ctx, g.Client(), g.Logger(), cloudInitCompleteFile, sshutil.TestRegularFile)
	if err != nil {
		return trace.Wrap(err)
	}

	return trace.Wrap(err)
}

func connectVMs(ctx context.Context, log logrus.FieldLogger, params cloudDynamicParams, nodes []infra.Node) (out []*gravity, err error) {
	errC := make(chan error, len(nodes))
	nodeC := make(chan interface{}, len(nodes))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, node := range nodes {
		go func(node infra.Node) {
			gnode, err := connectVM(ctx, log, node, params)
			nodeC <- gnode
			errC <- err
		}(node)
	}

	gnodes, err := utils.Collect(ctx, cancel, errC, nodeC)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	for _, node := range gnodes {
		out = append(out, node.(*gravity))
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Node().PrivateAddr() < out[j].Node().PrivateAddr()
	})

	return out, nil
}

func connectVM(ctx context.Context, log logrus.FieldLogger, node infra.Node, param cloudDynamicParams) (*gravity, error) {
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
	return g, nil
}

// ConfigureNode is used to configure a provisioned node
// 1. wait for node to boot
// 2. (TODO) run bootstrap scripts - as Azure doesn't support them for RHEL/CentOS, will migrate here
// 2.  - i.e. run bootstrap commands, load installer, etc.
// TODO: migrate bootstrap scripts here as well;
func configureVM(ctx context.Context, log logrus.FieldLogger, node *gravity, param cloudDynamicParams) (err error) {
	switch param.CloudProvider {
	case constants.AWS:
		err = bootstrapCloud(ctx, node, param)
	case constants.Azure:
		err = bootstrapAzure(ctx, node, param)
	case constants.GCE:
		err = bootstrapCloud(ctx, node, param)
	case constants.Ops, constants.Libvirt:
		// For ops installs the installer is not needed
	default:
		return trace.BadParameter("unsupported cloud provider %s", param.CloudProvider)
	}
	if err != nil {
		return trace.Wrap(err)
	}

	return nil
}

func validateDiskSpeed(ctx context.Context, nodes []*gravity, device string, logger logrus.FieldLogger) error {
	logger.Debug("Ensuring disk speed is adequate across nodes.")
	ctx, cancel := context.WithTimeout(ctx, diskWaitTimeout)
	defer cancel()
	err := waitDisks(ctx, nodes, []string{"/iotest", device}, logger)
	if err != nil {
		err = trace.Wrap(err, "VM disks did not meet performance requirements, tear down as non-usable")
		logger.WithError(err).Error("VM disks did not meet performance requirements, tear down as non-usable.")
		return trace.Wrap(err)
	}
	return nil
}

// waitDisks is a necessary workaround for Azure VMs to wait until their disk initialization processes are complete
// otherwise it'll fail telekube pre-install checks
func waitDisks(ctx context.Context, nodes []*gravity, paths []string, logger logrus.FieldLogger) error {
	errs := make(chan error, len(nodes))

	for _, node := range nodes {
		go func(node *gravity) {
			errs <- waitDisk(ctx, node, paths, minDiskSpeed, logger)
		}(node)
	}

	return trace.Wrap(utils.CollectErrors(ctx, errs))
}

// waitDisk will wait specific disk performance to report OK
func waitDisk(ctx context.Context, node *gravity, paths []string, minSpeed uint64, logger logrus.FieldLogger) error {
	err := wait.Retry(ctx, func() error {
		for _, path := range paths {
			if !strings.HasPrefix(path, "/dev") {
				defer func() {
					errRemove := sshutil.Run(ctx, node.Client(), node.Logger(),
						fmt.Sprintf("sudo /bin/rm -f %s", path), nil)
					if errRemove != nil {
						logger.Warnf("Failed to remove path: %v.", errRemove)
					}
				}()
			}
			var out string
			err := sshutil.RunAndParse(ctx, node.Client(), node.Logger(),
				fmt.Sprintf("sudo dd if=/dev/zero of=%s bs=100K count=1024 conv=fdatasync 2>&1", path),
				nil, sshutil.ParseAsString(&out))
			if err != nil {
				return wait.Abort(trace.Wrap(err))
			}
			speed, err := ParseDDOutput(out)
			if err != nil {
				return wait.Abort(trace.Wrap(err))
			}
			if speed < minSpeed {
				return wait.Continue("%s has %v/s < minimum of %v/s",
					path, humanize.Bytes(speed), humanize.Bytes(minSpeed))
			}
		}
		return nil
	})
	return trace.Wrap(err)
}

// destroyResource executes the specified destroy handler using
// default context
func destroyResource(handler func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), finalTeardownTimeout)
	defer cancel()
	return trace.Wrap(handler(ctx))
}

// Cluster describes the result of provisioning cluster infrastructure.
type Cluster struct {
	// Nodes is the list of gravity nodes in the cluster
	Nodes []Gravity
	// Destroy is the resource destruction handler
	Destroy DestroyFn
}
