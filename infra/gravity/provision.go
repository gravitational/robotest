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
	fromConfig   *ProvisionerConfig
	dockerDevice string
	user         string
	os           string
	installDir   string
	installerUrl string
	tf           terraform.Config
	env          map[string]string
}

// makeDynamicParams takes base config, validates it and returns cloudDynamicParams
func makeDynamicParams(t *testing.T, baseConfig *ProvisionerConfig) *cloudDynamicParams {
	require.NotNil(t, baseConfig)

	param := &cloudDynamicParams{fromConfig: baseConfig}

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

	param.installerUrl = baseConfig.InstallerURL
	require.NotEmpty(t, param.installerUrl, "InstallerUrl")

	param.installDir = filepath.Join("/home", param.user, "install")

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

		param.dockerDevice = param.tf.AWS.DockerDevice

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

		param.dockerDevice = param.tf.Azure.DockerDevice
	}

	require.NotEmpty(t, param.dockerDevice, "docker device")
	return param
}

// Provision gets VMs up, running and ready to use
func Provision(ctx context.Context, t *testing.T, baseConfig *ProvisionerConfig) ([]Gravity, DestroyFn, error) {
	validateConfig(t, baseConfig)
	params := makeDynamicParams(t, baseConfig)

	logFn := utils.Logf(t, baseConfig.Tag())
	logFn("[provision] OS=%s NODES=%d TAG=%s DIR=%s", baseConfig.os, baseConfig.nodeCount, baseConfig.tag, baseConfig.stateDir)

	p, err := terraform.New(filepath.Join(baseConfig.stateDir, "tf"), params.tf)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	_, err = p.Create(false)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	resourceAllocated(baseConfig.CloudProvider, baseConfig.Tag())
	destroyFn := wrapDestroyFn(baseConfig.Tag(), p.Destroy)

	nodes := p.NodePool().Nodes()
	type result struct {
		gravity Gravity
		err     error
	}
	resultCh := make(chan *result, len(nodes))

	for _, node := range nodes {
		go func(n infra.Node) {
			var res result
			res.gravity, res.err = PrepareGravity(ctx, t, n, params)
			resultCh <- &res
		}(node)
	}

	gravityNodes := []Gravity{}
	errors := []error{}
	for range nodes {
		select {
		case <-ctx.Done():
			return nil, nil, trace.Errorf("timed out waiting for nodes to provision")
		case res := <-resultCh:
			if res.err != nil {
				errors = append(errors, res.err)
			} else {
				gravityNodes = append(gravityNodes, res.gravity)
			}
		}
	}

	if len(errors) != 0 {
		aggError := trace.NewAggregate(errors...)
		logFn("[provision] cleanup after error provisioning : %v", aggError)

		destroyFn(ctx, t)
		return nil, destroyFn, aggError
	}

	logFn("[provision] OS=%s NODES=%d TAG=%s DIR=%s", baseConfig.os, baseConfig.nodeCount, baseConfig.tag, baseConfig.stateDir)
	for _, node := range gravityNodes {
		logFn("\t%v", node)
	}

	return sorted(gravityNodes), destroyFn, nil
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
	cloudInitCompleteFile = "/var/lib/bootstrap_complete"
)

const cloudInitWait = time.Second * 10

// Run bootstrap scripts on a node
func bootstrap(ctx context.Context, g Gravity, param *cloudDynamicParams) error {
	err := sshutil.Run(ctx, g, "sudo whoami", nil)
	if err != nil {
		return trace.Wrap(err, "sudo check")
	}

	// TODO: implement simple line-by-line execution of existing .sh scripts
	// Azure doesn't support cloud-init for RHEL/CentOS so need migrate them here
	return sshutil.WaitForFile(ctx, g, cloudInitCompleteFile, sshutil.TestRegularFile, cloudInitWait)
}

// ConfigureNode is used to configure a provisioned node
// 1. wait for node to boot
// 2. (TODO) run bootstrap scripts - as Azure doesn't support them for RHEL/CentOS, will migrate here
// 2.  - i.e. run bootstrap commands, load installer, etc.
// TODO: migrate bootstrap scripts here as well;
func PrepareGravity(ctx context.Context, t *testing.T, node infra.Node, param *cloudDynamicParams) (Gravity, error) {
	g := &gravity{
		node:         node,
		installDir:   param.installDir,
		dockerDevice: param.dockerDevice,
		logFn:        t.Logf,
		fromConfig:   param.fromConfig,
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

	configCompleteFile := filepath.Join(param.installDir, "config_complete")

	err = sshutil.TestFile(ctx, g, configCompleteFile, sshutil.TestRegularFile)

	if err == nil {
		g.Logf("already configured")
		return g, nil
	}

	if !trace.IsNotFound(err) {
		return nil, trace.Wrap(err)
	}

	g.Logf("Transferring installer from %s ...", param.installerUrl)
	tgz, err := sshutil.TransferFile(ctx, g, param.installerUrl, param.installDir, param.env)
	if err != nil {
		g.Logf("Failed to transfer installer %s : %v", param.installerUrl, err)
		return nil, trace.Wrap(err, param.installerUrl)
	}

	err = sshutil.RunCommands(ctx, g, []sshutil.Cmd{
		{fmt.Sprintf("tar -xvf %s -C %s", tgz, param.installDir), nil},
		{fmt.Sprintf("touch %s", configCompleteFile), nil},
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return g, nil
}
