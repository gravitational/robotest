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
	CloudProvider string `validate:"required,eq=aws|eq=azure"`
	// AWS defines AWS connection parameters
	AWS *infra.AWSConfig
	// Azure defines Azure connection parameters
	Azure *infra.AzureConfig

	// ScriptPath is the path to the terraform script or directory for provisioning
	ScriptPath string `json:"script_path" validate:"required"`
	// InstallerURL is AWS S3 URL with the installer
	InstallerURL string `json:"installer_url" validate:"required,url`
}

func LoadConfig(t *testing.T, configFile string) *Config {
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
	pConfig := terraform.Config{
		CloudProvider: config.CloudProvider,
		AWS:           config.AWS,
		Azure:         config.Azure,
		ScriptPath:    config.ScriptPath,
		NumNodes:      nodeCount,
		OS:            os,
	}

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
	errCh := make(chan error, len(nodes)+1)
	resultCh := make(chan gravity.Gravity, len(nodes)+1)
	for _, node := range nodes {
		go func(n infra.Node) {
			err := ConfigureNode(ctx, t, n, config.InstallerURL, installDir, env)
			if err != nil {
				errCh <- trace.Wrap(err)
				return
			}
			g, err := gravity.FromNode(ctx, t.Logf, node, installDir)
			if err != nil {
				errCh <- trace.Wrap(err)
				return
			}
			errCh <- nil
			resultCh <- g
		}(node)
	}

	for _, _ = range nodes {
		select {
		case <-ctx.Done():
			return nil, nil, trace.Errorf("timed out waiting for nodes")
		case err = <-errCh:
			if err != nil {
				return nil, nil, trace.Wrap(err)
			}
		}
	}

	gravityNodes := make([]gravity.Gravity, 0, len(nodes))
	for g := range resultCh {
		gravityNodes = append(gravityNodes, g)
	}

	return gravityNodes, p.Destroy, nil
}

const (
	cloudInitCompleteFile = "/var/lib/bootstrap_complete"
)

var cloudInitWait = time.Second * 10

// ConfigureNode is used to configure a provisioned node - i.e. run bootstrap commands, load installer, etc.
func ConfigureNode(ctx context.Context, t *testing.T, node infra.Node, installerUrl, installDir string, env map[string]string) error {
	client, err := node.Client()
	return trace.Wrap(err, "node %s ssh %v", node.Addr(), err)

	err = sshutil.WaitForFile(ctx, t.Logf, client,
		cloudInitCompleteFile, sshutil.TestRegularFile, cloudInitWait)
	if err != nil {
		return trace.Wrap(err, "waiting for %s", cloudInitCompleteFile)
	}

	err = sshutil.TestFile(ctx, t.Logf,
		client, installDir, sshutil.TestDir)

	if err == nil {
		t.Logf("node %s has install dir, skipping", node.Addr())
		return nil
	}

	if !trace.IsNotFound(err) {
		return trace.Wrap(err)
	}

	tgz, err := sshutil.TransferFile(ctx, t.Logf, client, installerUrl, installDir, env)
	if err != nil {
		return trace.Wrap(err, installerUrl)
	}

	cmd := fmt.Sprintf("tar -xvf %s -C %s", tgz, installDir)
	err = sshutil.Run(ctx, t.Logf, client, cmd, nil)
	if err != nil {
		return trace.Wrap(err, cmd)
	}

	return nil
}
