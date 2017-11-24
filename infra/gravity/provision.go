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
	"github.com/gravitational/robotest/infra/ops"
	"github.com/gravitational/robotest/infra/terraform"
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

// Provision will attempt to provision the requested cluster
func (c *TestContext) Provision(cfg ProvisionerConfig) ([]Gravity, DestroyFn, error) {
	var nodes []Gravity
	var destroy DestroyFn
	var err error
	switch cfg.CloudProvider {
	case "azure", "aws":
		nodes, destroy, err = c.provisionCloud(cfg)
	case "ops":
		nodes, destroy, err = c.provisionOps(cfg)
	default:
		nodes, destroy, err = nil, nil, trace.Wrap(errors.New("unknown cloud provider"))
	}

	// if we get an error during provision, let's try and call the DestroyFn if it's been provided to clean up any resources
	if err != nil && destroy != nil {
		dErr := destroy()
		if dErr != nil {
			err = trace.NewAggregate(err, dErr)
		}
	}

	return nodes, destroy, err
}

// provisionOps splits off the provisioning call flow to use an ops center and the provisioner specified by the app
// to provision the cluster to test
func (c *TestContext) provisionOps(cfg ProvisionerConfig) ([]Gravity, DestroyFn, error) {
	c.Logger().WithField("config", cfg).Debug("Provisioning via Ops Center")

	// ensure third aprty connections are working first, before wasting a bunch of time
	c.Logger().Debug("attempting to connect to AWS api")
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(cfg.Ops.Region),
		Credentials: credentials.NewStaticCredentials(cfg.Ops.AccessKey, cfg.Ops.SecretKey, ""),
	})
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	c.Logger().Debug("logging into the ops center")
	out, err := exec.Command("tele", "login", "-o", cfg.Ops.URL, "--key", cfg.Ops.OpsKey).Output()
	if err != nil {
		return nil, nil, trace.WrapWithMessage(err, string(out))
	}

	// generate a random cluster name
	clusterName := fmt.Sprint("robotest-", uuid.NewV4().String())
	c.Logger().Debug("Generated cluster name: ", clusterName)

	c.Logger().Debug("validating configuration")
	err = validateConfig(cfg)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	// first, we need to create a cluster from a template, and write it to a file that can be imported
	c.Logger().Debug("generating cluster.yaml for ops center")
	clusterPath := path.Join(cfg.StateDir, "cluster.yaml")
	err = os.MkdirAll(cfg.StateDir, 0777)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	defn, err := generateClusterDefn(cfg, clusterName)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	err = ioutil.WriteFile(clusterPath, []byte(defn), 0644)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	// next, we need to tell the ops center to create our cluster
	c.Logger().Debug("requesting ops center to provision our cluster")
	out, err = exec.Command("tele", "create", clusterPath).Output()
	if err != nil {
		return nil, nil, trace.WrapWithMessage(err, string(out))
	}

	// now that we've requested cluster creation, let's setup our destroyFn so that we can clean up after ourselves
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
				return nil, destroyFn, trace.Wrap(fmt.Errorf("unexpected cluster status: %v", status))
			}
		}
	}

	// now that the requested cluster has been created, we have to build the []Gravity slice of nodes
	ec2svc := ec2.New(sess)
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:KubernetesCluster"),
				Values: []*string{aws.String(clusterName)},
			},
		},
	}
	c.Logger().Debug("attempting to get a listing of instances from AWS for the cluster")
	resp, err := ec2svc.DescribeInstances(params)
	if err != nil {
		return nil, destroyFn, trace.Wrap(err)
	}

	ctx, cancel := context.WithTimeout(c.Context(), cloudInitTimeout)
	defer cancel()
	gravityNodes := []Gravity{}
	for _, reservation := range resp.Reservations {
		for _, inst := range reservation.Instances {
			c.Logger().Debugf("building new node for testing. public_ip: %v private_ip: %v ssh_user: %v ssh_key_path: %v",
				*inst.PublicIpAddress, *inst.PrivateIpAddress, cfg.Ops.SSHUser, cfg.Ops.SSHKeyPath)
			node := ops.New(*inst.PublicIpAddress, *inst.PrivateIpAddress, cfg.Ops.SSHUser, cfg.Ops.SSHKeyPath)
			cloudParams, err := makeDynamicParams(cfg)
			if err != nil {
				return nil, destroyFn, trace.Wrap(err)
			}
			c.Logger().Debug("configureVM on gravity node")
			gravityNode, err := configureVM(ctx, c.Logger(), node, *cloudParams)
			if err != nil {
				return nil, destroyFn, trace.Wrap(err)
			}
			gravityNodes = append(gravityNodes, gravityNode)
		}
	}

	c.Logger().Debugf("running post provisioning tasks")
	err = c.postProvision(cfg, gravityNodes)
	if err != nil {
		return nil, destroyFn, trace.Wrap(err)
	}

	c.Logger().Debug("Ensuring disk speed is adequate across nodes")
	ctx, cancel = context.WithTimeout(c.Context(), diskWaitTimeout)
	defer cancel()
	err = waitDisks(ctx, gravityNodes, []string{"/iotest", path.Join(cfg.dockerDevice, "/iotest")})
	if err != nil {
		err = trace.Wrap(err, "VM disks do not meet minimum write performance requirements")
		c.Logger().WithError(err).Error(err.Error())
		return nil, destroyFn, err
	}

	c.Logger().Info("provisioning complete")
	return gravityNodes, destroyFn, nil
}

// Provision gets VMs up, running and ready to use
func (c *TestContext) provisionCloud(cfg ProvisionerConfig) ([]Gravity, DestroyFn, error) {
	c.Logger().WithField("config", cfg).Debug("Provisioning VMs")

	err := validateConfig(cfg)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	nodes, destroyFn, params, err := runTerraform(c.Context(), cfg, c.Logger())
	if err != nil {
		return nil, nil, trace.Wrap(err)
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

	c.Logger().Debug("Ensuring disk speed is adequate across nodes")
	ctx, cancel = context.WithTimeout(c.Context(), diskWaitTimeout)
	defer cancel()
	err = waitDisks(ctx, gravityNodes, []string{"/iotest", cfg.dockerDevice})
	if err != nil {
		err = trace.Wrap(err, "VM disks do not meet minimum write performance requirements")
		c.Logger().WithError(err).Error(err.Error())
		return nil, nil, err
	}

	c.Logger().WithField("nodes", gravityNodes).Debug("Provisioning complete")

	return gravityNodes, wrapDestroyFn(c, cfg.Tag(), gravityNodes, destroyFn), nil
}

func (c *TestContext) postProvision(cfg ProvisionerConfig, gravityNodes []Gravity) error {
	c.Logger().Debug("Streaming logs")
	for _, node := range gravityNodes {
		go node.(*gravity).streamLogs(c.Context())
	}

	ctx, cancel := context.WithTimeout(c.Context(), clockSyncTimeout)
	defer cancel()

	c.Logger().Debug("Synchronizing clocks")
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
	case "aws":
		err = bootstrapAWS(ctx, g, param)
	case "azure":
		err = bootstrapAzure(ctx, g, param)
	case "ops":

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
