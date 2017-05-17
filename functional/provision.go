package functional

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/robotest/infra/terraform"
	sshutil "github.com/gravitational/robotest/lib/ssh"

	"github.com/gravitational/trace"

	"github.com/go-yaml/yaml"
	"github.com/stretchr/testify/require"
	"gopkg.in/go-playground/validator.v9"
)

type Config struct {
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

func LoadConfig(t *testing.T, configFile string) *Config {
	require.NotEmpty(t, configFile, "config file")
	f, err := os.Open(configFile)
	require.NoError(t, err, configFile)
	defer f.Close()

	configBytes, err := ioutil.ReadAll(f)
	require.NoError(t, err)

	cfg := Config{}
	err = yaml.Unmarshal(configBytes, &cfg)
	require.NoError(t, err, configFile)

	err = validator.New().Struct(&cfg)
	if err == nil {
		return &cfg
	}

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			t.Errorf("   field %s=%v fails rule %s", fieldError.Field(), fieldError.Value(), fieldError.Tag())
		}
	}

	t.Fatalf("Configuration file %s has errors", configFile)
	// never reached
	return nil
}

type DestroyFn func() error

// Provision gets VMs up, running and ready to use
func Provision(ctx context.Context, t *testing.T, config *Config, tag, stateDir string, nodeCount int, os string) ([]gravity.Gravity, DestroyFn, error) {
	aws := *config.AWS
	azure := *config.Azure

	aws.ClusterName = tag
	azure.ResourceGroup = tag

	pConfig := terraform.Config{
		CloudProvider: config.CloudProvider,
		AWS:           &aws,
		Azure:         &azure,
		ScriptPath:    config.ScriptPath,
		NumNodes:      nodeCount,
		OS:            os,
	}

	logFn := Logf(t, "provision")
	logFn("Starting Terraform tag=%s, stateDir=%s, nodes=%d, os=%s", tag, stateDir, nodeCount, os)
	p, err := terraform.New(filepath.Join(stateDir, tag, "tf"), pConfig)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	_, err = p.Create(false)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	user, _ := p.SSHConfig()
	installDir := filepath.Join("/home", user, "install")

	var env map[string]string = nil
	if config.AWS != nil {
		env = map[string]string{
			"AWS_ACCESS_KEY_ID":     config.AWS.AccessKey,
			"AWS_SECRET_ACCESS_KEY": config.AWS.SecretKey,
			"AWS_DEFAULT_REGION":    config.AWS.Region,
		}
	}

	nodes := p.NodePool().Nodes()
	errCh := make(chan error, len(nodes))
	resultCh := make(chan gravity.Gravity, len(nodes))
	for _, node := range nodes {
		go func(n infra.Node) {
			g, err := ConfigureNode(ctx, logFn, n, config.InstallerURL, installDir, env)
			resultCh <- g
			errCh <- trace.Wrap(err)
		}(node)
	}

	gravityNodes := make([]gravity.Gravity, 0, len(nodes))
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

// ConfigureNode is used to configure a provisioned node - i.e. run bootstrap commands, load installer, etc.
func ConfigureNode(ctx context.Context, logFn sshutil.LogFnType, node infra.Node, installerUrl, installDir string, env map[string]string) (gravity.Gravity, error) {
	g, err := gravity.FromNode(ctx, logFn, node, installDir)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = sshutil.WaitForFile(ctx, logFn, g.Client(),
		cloudInitCompleteFile, sshutil.TestRegularFile, cloudInitWait)
	if err != nil {
		return nil, trace.Wrap(err, "waiting for %s", cloudInitCompleteFile)
	}

	configCompleteFile := filepath.Join(installDir, "config_complete")

	err = sshutil.TestFile(ctx, logFn,
		g.Client(), configCompleteFile, sshutil.TestRegularFile)

	if err == nil {
		logFn("node %v was already configured", node.Addr())
		return g, nil
	}

	if !trace.IsNotFound(err) {
		return nil, trace.Wrap(err)
	}

	tgz, err := sshutil.TransferFile(ctx, logFn, g.Client(), installerUrl, installDir, env)
	if err != nil {
		return nil, trace.Wrap(err, installerUrl)
	}

	err = sshutil.RunCommands(ctx, logFn, g.Client(), []sshutil.Cmd{
		{fmt.Sprintf("tar -xvf %s -C %s", tgz, installDir), nil},
		{fmt.Sprintf("touch %s", configCompleteFile), nil},
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return g, nil
}
