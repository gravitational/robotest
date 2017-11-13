package gravity

import (
	"context"
	"fmt"
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

	"github.com/gravitational/trace"

	"github.com/dustin/go-humanize"
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

// RestoreCheckpoint
func (c *TestContext) RestoreCheckpoint(cfg ProvisionerConfig, checkpoint string, param interface{}) ([]Gravity, error) {
	/*
		if c.suite.imageRegistry == nil {
			return nil, trace.NotFound("image registry service unavailable")
		}
	*/
	cfg.Azure = &(*cfg.Azure)
	cfg.FromImage = &infra.VmImage{
		Region:        "westus",
		ResourceGroup: "R293-INSTALL-1-REDHAT7.4-OVERLAY2-3N",
	}
	/*
		cfg.FromImage, err = c.suite.imageRegistry.Locate(c.Context(), checkpoint, param)
		if err != nil {
			if trace.IsNotFound(err) {
				return trace.Wrap(err)
			}
			c.Logger().WithError(err).Warn("Azure image locator failed for %q/%v", checkpoint, param)
			return trace.NotFound("Image locator internal error")
		}
	*/
	return c.Provision(cfg)
}

// Provision gets VMs up, running and ready to use
func (c *TestContext) Provision(cfg ProvisionerConfig) ([]Gravity, error) {
	c.Logger().WithField("config", cfg).Debug("Provisioning VMs")

	err := validateConfig(cfg)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	nodes, destroyFn, params, err := runTerraform(c.Context(), cfg, c.Logger())
	if err != nil {
		return nil, trace.Wrap(err)
	}

	ctx, cancel := context.WithTimeout(c.Context(), cloudInitTimeout)
	defer cancel()

	c.Logger().Debug("Configuring VMs")
	gravityNodes, err := configureVMs(ctx, c.Logger(), *params, nodes)
	if err != nil {
		c.Logger().WithError(err).Error("some nodes initialization failed, teardown this setup as non-usable")
		return nil, trace.NewAggregate(err, destroyFn(ctx))
	}

	c.Logger().Debug("Streaming logs")
	for _, node := range gravityNodes {
		go node.(*gravity).streamLogs(c.Context())
	}

	ctx, cancel = context.WithTimeout(c.Context(), clockSyncTimeout)
	defer cancel()

	c.Logger().Debug("Synchronizing clocks")
	timeNodes := []sshutil.SshNode{}
	for _, node := range gravityNodes {
		timeNodes = append(timeNodes, sshutil.SshNode{node.Client(), node.Logger()})
	}
	if err := sshutil.WaitTimeSync(ctx, timeNodes); err != nil {
		return nil, trace.NewAggregate(err, destroyFn(ctx))
	}

	if cfg.waitDisks {
		c.Logger().Debug("Ensuring disk speed is adequate across nodes")
		ctx, cancel = context.WithTimeout(c.Context(), diskWaitTimeout)
		defer cancel()
		err = waitDisks(ctx, gravityNodes, []string{"/iotest", cfg.dockerDevice})
		if err != nil {
			err = trace.Wrap(err, "VM disks do not meet minimum write performance requirements")
			c.Logger().WithError(err).Error(err.Error())
			return nil, err
		}
	}

	c.Logger().WithField("nodes", gravityNodes).Debug("Provisioning complete")

	c.resourceDestroyFuncs = append(c.resourceDestroyFuncs,
		wrapDestroyFn(c, cfg.Tag(), gravityNodes, destroyFn))
	return gravityNodes, nil
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
