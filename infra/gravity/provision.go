package gravity

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/terraform"
	sshutil "github.com/gravitational/robotest/lib/ssh"

	"github.com/gravitational/trace"

	"github.com/go-yaml/yaml"
	"github.com/stretchr/testify/require"
	"gopkg.in/go-playground/validator.v9"
)

type ProvisionerConfig struct {
	// DeployTo defines cloud to deploy to
	CloudProvider string `yaml:"cloud_provider" validate:"required,eq=aws|eq=azure"`
	// AWS defines AWS connection parameters
	AWS *infra.AWSConfig `yaml:"aws"`
	// Azure defines Azure connection parameters
	Azure *infra.AzureConfig `yaml:"azure"`

	// ScriptPath is the path to the terraform script or directory for provisioning
	ScriptPath string `yaml:"script_path" validate:"required"`
	// InstallerURL is AWS S3 URL with the installer
	InstallerURL string `yaml:"installer_url" validate:"required,url`
}

func LoadConfig(t *testing.T, configFile string) *ProvisionerConfig {
	require.NotEmpty(t, configFile, "config file")
	f, err := os.Open(configFile)
	require.NoError(t, err, configFile)
	defer f.Close()

	configBytes, err := ioutil.ReadAll(f)
	require.NoError(t, err)

	cfg := ProvisionerConfig{}
	err = yaml.Unmarshal(configBytes, &cfg)
	require.NoError(t, err, configFile)

	err = validator.New().Struct(&cfg)
	if err == nil {
		return &cfg
	}

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		t.Errorf("Config %s has errors:", configFile)
		for _, fieldError := range validationErrors {
			t.Errorf(" * %s=\"%v\" fails \"%s\"", fieldError.Field(), fieldError.Value(), fieldError.Tag())
		}
		require.FailNow(t, "Fix config")
	}

	// never reached
	return nil
}

// function which will destroy previously created remote resources
type DestroyFn func() error

// scheduleDestroy will register resource (placement) group for destruction with external service
// based on Context expiration deadline. This would only be used in continuous operation (i.e. part of CLI)
func scheduleDestroy(ctx context.Context, cloud, tag string) {
	// TODO: implement
}

// cloudDynamicParams is a necessary evil to marry terraform vars, e2e legacy objects and needs of this provisioner
type cloudDynamicParams struct {
	dockerDevice string
	user         string
	os           string
	installDir   string
	installerUrl string
	tf           terraform.Config
	env          map[string]string
}

// makeDynamicParams takes base config, validates it and returns TerraformConfig
func makeDynamicParams(t *testing.T, baseConfig *ProvisionerConfig, tag, os string, nodeCount int) *cloudDynamicParams {
	require.NotNil(t, baseConfig)

	param := &cloudDynamicParams{}

	// OS name is cloud-init script specific
	// enforce compatible values
	var ok bool
	osUsernames := map[string]string{
		"ubuntu": "robotest",
		"debian": "admin",
		"rhel":   "redhat", // TODO: check
		"centos": "centos",
	}

	param.user, ok = osUsernames[os]
	require.True(t, ok, os)

	param.installerUrl = baseConfig.InstallerURL
	require.NotEmpty(t, param.installerUrl, "InstallerUrl")

	param.installDir = filepath.Join("/home", param.user, "install")

	param.tf = terraform.Config{
		CloudProvider: baseConfig.CloudProvider,
		ScriptPath:    baseConfig.ScriptPath,
		NumNodes:      nodeCount,
		OS:            os,
	}

	if baseConfig.AWS != nil {
		param.tf.AWS = &(*baseConfig.AWS)
		param.tf.AWS.ClusterName = tag
		param.tf.AWS.SSHUser = param.user

		param.dockerDevice = param.tf.AWS.DockerDevice

		param.env = map[string]string{
			"AWS_ACCESS_KEY_ID":     param.tf.AWS.AccessKey,
			"AWS_SECRET_ACCESS_KEY": param.tf.AWS.SecretKey,
			"AWS_DEFAULT_REGION":    param.tf.AWS.Region,
		}
	}

	if baseConfig.Azure != nil {
		param.tf.Azure = &(*baseConfig.Azure)
		param.tf.Azure.ResourceGroup = tag
		param.tf.Azure.SSHUser = param.user

		param.dockerDevice = param.tf.Azure.DockerDevice
	}

	require.NotEmpty(t, param.dockerDevice, "docker device")
	return param
}

// Provision gets VMs up, running and ready to use
func Provision(ctx context.Context, t *testing.T, baseConfig *ProvisionerConfig, tag, stateDir string, nodeCount int, os string) ([]Gravity, DestroyFn, error) {
	params := makeDynamicParams(t, baseConfig, tag, os, nodeCount)

	logFn := Logf(t, "provision")
	logFn("Terraform tag=%s, stateDir=%s, nodes=%d, os=%s", tag, stateDir, nodeCount, os)
	p, err := terraform.New(filepath.Join(stateDir, tag, "tf"), params.tf)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	scheduleDestroy(ctx, baseConfig.CloudProvider, tag)

	_, err = p.Create(false)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	nodes := p.NodePool().Nodes()
	errCh := make(chan error, len(nodes))
	resultCh := make(chan Gravity, len(nodes))
	for _, node := range nodes {
		go func(n infra.Node) {
			g, err := PrepareGravity(ctx, logFn, n, params)
			resultCh <- g
			errCh <- trace.Wrap(err)
		}(node)
	}

	gravityNodes := make([]Gravity, 0, len(nodes))
	errors := []error{}
	for i := 0; i < 2*len(nodes); i++ {
		select {
		case <-ctx.Done():
			return nil, nil, trace.Errorf("timed out waiting for nodes to provision")
		case err = <-errCh:
			if err != nil {
				errors = append(errors, err)
			}
		case g := <-resultCh:
			gravityNodes = append(gravityNodes, g)
		}
	}

	if len(errors) != 0 {
		return nil, p.Destroy, trace.NewAggregate(errors...)
	}

	logFn("Provisioned %d nodes, OS=%s, tag=%s, stateDir=%s", len(gravityNodes), os, tag, stateDir)
	return gravityNodes, p.Destroy, nil
}

const (
	cloudInitCompleteFile = "/var/lib/bootstrap_complete"
)

var cloudInitWait = time.Second * 10

// Run bootstrap scripts on a node
func bootstrap(ctx context.Context, logFn sshutil.LogFnType, client *ssh.Client, param *cloudDynamicParams) error {
	// TODO: implement simple line-by-line execution of existing .sh scripts
	// Azure doesn't support cloud-init for RHEL/CentOS so need migrate them here
	return sshutil.WaitForFile(ctx, logFn, client,
		cloudInitCompleteFile, sshutil.TestRegularFile, cloudInitWait)
}

// ConfigureNode is used to configure a provisioned node
// 1. wait for node to boot
// 2. (TODO) run bootstrap scripts - as Azure doesn't support them for RHEL/CentOS, will migrate here
// 2.  - i.e. run bootstrap commands, load installer, etc.
// TODO: migrate bootstrap scripts here as well;
func PrepareGravity(ctx context.Context, logFn sshutil.LogFnType, node infra.Node, param *cloudDynamicParams) (Gravity, error) {
	g, err := fromNode(ctx, logFn, node, param.installDir, param.dockerDevice)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = bootstrap(ctx, logFn, g.Client(), param)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	configCompleteFile := filepath.Join(param.installDir, "config_complete")

	err = sshutil.TestFile(ctx, logFn,
		g.Client(), configCompleteFile, sshutil.TestRegularFile)

	if err == nil {
		logFn("node %v was already configured", node.Addr())
		return g, nil
	}

	if !trace.IsNotFound(err) {
		return nil, trace.Wrap(err)
	}

	tgz, err := sshutil.TransferFile(ctx, logFn, g.Client(),
		param.installerUrl, param.installDir, param.env)
	if err != nil {
		return nil, trace.Wrap(err, param.installerUrl)
	}

	err = sshutil.RunCommands(ctx, logFn, g.Client(), []sshutil.Cmd{
		{fmt.Sprintf("tar -xvf %s -C %s", tgz, param.installDir), nil},
		{fmt.Sprintf("touch %s", configCompleteFile), nil},
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return g, nil
}
