package gravity

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/terraform"
	sshutil "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/utils"

	"github.com/gravitational/trace"

	"github.com/stretchr/testify/require"
)

// cloudDynamicParams is a necessary evil to marry terraform vars, e2e legacy objects and needs of this provisioner
type cloudDynamicParams struct {
	ProvisionerConfig
	user    string
	homeDir string
	tf      terraform.Config
	env     map[string]string
}

// makeDynamicParams takes base config, validates it and returns cloudDynamicParams
func makeDynamicParams(t *testing.T, baseConfig ProvisionerConfig) cloudDynamicParams {
	require.NotNil(t, baseConfig)

	param := cloudDynamicParams{ProvisionerConfig: baseConfig}

	// OS name is cloud-init script specific
	// enforce compatible values
	var ok bool
	osUsernames := map[string]string{
		"ubuntu": "robotest",
		"debian": "admin",
		"rhel":   "redhat", // TODO: check
		"centos": "centos",
	}

	param.user, ok = osUsernames[baseConfig.os]
	require.True(t, ok, baseConfig.os)

	param.homeDir = filepath.Join("/home", param.user)

	param.tf = terraform.Config{
		CloudProvider: baseConfig.CloudProvider,
		ScriptPath:    baseConfig.ScriptPath,
		NumNodes:      int(baseConfig.nodeCount),
		OS:            baseConfig.os,
	}

	if baseConfig.AWS != nil {
		aws := *baseConfig.AWS
		param.tf.AWS = &aws
		param.tf.AWS.ClusterName = baseConfig.tag
		param.tf.AWS.SSHUser = param.user

		param.env = map[string]string{
			"AWS_ACCESS_KEY_ID":     param.tf.AWS.AccessKey,
			"AWS_SECRET_ACCESS_KEY": param.tf.AWS.SecretKey,
			"AWS_DEFAULT_REGION":    param.tf.AWS.Region,
		}
	}

	if baseConfig.Azure != nil {
		azure := *baseConfig.Azure
		param.tf.Azure = &azure
		param.tf.Azure.ResourceGroup = baseConfig.tag
		param.tf.Azure.SSHUser = param.user
	}

	return param
}

func configureVMs(baseCtx context.Context, t *testing.T, params cloudDynamicParams, nodes []infra.Node) ([]Gravity, error) {
	errChan := make(chan error, len(nodes))
	nodeChan := make(chan interface{}, len(nodes))

	ctx, cancel := context.WithCancel(baseCtx)
	defer cancel()

	for _, node := range nodes {
		go func(node infra.Node) {
			val, err := configureVM(ctx, t, node, params)
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

// Provision gets VMs up, running and ready to use
func Provision(ctx context.Context, t *testing.T, baseConfig ProvisionerConfig) ([]Gravity, DestroyFn, error) {
	validateConfig(t, baseConfig)
	params := makeDynamicParams(t, baseConfig)

	logFn := utils.Logf(t, baseConfig.Tag())

	logFn("(1/3) Provisioning VMs")
	nodes, destroyFn, err := runTerraform(ctx, baseConfig, params)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	logFn("(2/3) Configuring VMs")
	gravityNodes, err := configureVMs(ctx, t, params, nodes)
	if err != nil {
		logFn("some nodes initialization failed, teardown this setup as non-usable")
		return nil, nil, trace.NewAggregate(err, destroyFn(ctx))
	}

	logFn("(3/3) Synchronizing clocks")
	timeNodes := []sshutil.SshNode{}
	for _, node := range gravityNodes {
		timeNodes = append(timeNodes, node)
	}
	if err := sshutil.WaitTimeSync(ctx, timeNodes); err != nil {
		return nil, nil, trace.NewAggregate(err, destroyFn(ctx))
	}

	logFn("OS=%s NODES=%d TAG=%s DIR=%s", baseConfig.os, baseConfig.nodeCount, baseConfig.tag, baseConfig.stateDir)
	for _, node := range gravityNodes {
		logFn("\t%v", node)
	}

	return gravityNodes, wrapDestroyFn(baseConfig.Tag(), gravityNodes, destroyFn), nil
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

// Run bootstrap scripts on a node
func bootstrap(ctx context.Context, g Gravity, param cloudDynamicParams) error {
	err := sshutil.WaitForFile(ctx, g, waagentProvisionFile, sshutil.TestRegularFile, cloudInitWait)
	if err != nil {
		return trace.Wrap(err)
	}

	err = sshutil.TestFile(ctx, g, cloudInitCompleteFile, sshutil.TestRegularFile)
	if err == nil {
		g.Logf("node already bootstrapped")
		return nil
	}
	if !trace.IsNotFound(err) {
		return trace.Wrap(err)
	}

	err = sshutil.TestFile(ctx, g, cloudInitSupportedFile, sshutil.TestRegularFile)
	if err == nil {
		g.Logf("cloud-init underway")
		return sshutil.WaitForFile(ctx, g, cloudInitCompleteFile, sshutil.TestRegularFile, cloudInitWait)
	}
	if !trace.IsNotFound(err) {
		return trace.Wrap(err)
	}

	// apparently cloud-init scripts are not supported for given OS
	err = sshutil.RunScript(ctx, g,
		filepath.Join(param.ScriptPath, "bootstrap", fmt.Sprintf("%s.sh", param.os)),
		sshutil.SUDO)
	return trace.Wrap(err)
}

// ConfigureNode is used to configure a provisioned node
// 1. wait for node to boot
// 2. (TODO) run bootstrap scripts - as Azure doesn't support them for RHEL/CentOS, will migrate here
// 2.  - i.e. run bootstrap commands, load installer, etc.
// TODO: migrate bootstrap scripts here as well;
func configureVM(ctx context.Context, t *testing.T, node infra.Node, param cloudDynamicParams) (Gravity, error) {
	g := &gravity{
		node:  node,
		param: param,
		logFn: t.Logf,
		ts:    time.Now(),
	}

	client, err := sshClient(ctx, g.Logf, g.node)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	g.ssh = client

	err = bootstrap(ctx, g, param)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return g, nil
}
