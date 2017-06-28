package gravity

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/gravitational/robotest/infra"

	"github.com/go-yaml/yaml"
	"github.com/stretchr/testify/require"
	"gopkg.in/go-playground/validator.v9"
)

// ProvisionerConfig defines parameters required to provision hosts
// CloudProvider, AWS, Azure, ScriptPath and InstallerURL
type ProvisionerConfig struct {
	// DeployTo defines cloud to deploy to
	CloudProvider string `yaml:"cloud" validate:"required,eq=aws|eq=azure"`
	// AWS defines AWS connection parameters
	AWS *infra.AWSConfig `yaml:"aws"`
	// Azure defines Azure connection parameters
	Azure *infra.AzureConfig `yaml:"azure"`

	// ScriptPath is the path to the terraform script or directory for provisioning
	ScriptPath string `yaml:"script_path" validate:"required"`
	// InstallerURL is AWS S3 URL with the installer
	InstallerURL string `yaml:"installer_url" validate:"required,url`
	// StateDir defines base directory where to keep state (i.e. terraform configs/vars)
	StateDir string `yaml:"state_dir" validate:"required"`

	// Tag will group provisioned resources under for easy removal afterwards
	tag string `validate:"required"`
	// NodeCount defines amount of nodes to be provisioned
	nodeCount uint `validate:"gte=1"`
	// OS defines one of supported operating systems
	os string `validate:"required,eq=ubuntu|eq=debian|eq=rhel|eq=centos"`
	// dockerStorageDriver defines Docker storage driver
	storageDriver string `validate:"required,eq=overlay|overlay2|devicemapper|loopback"`
	// dockerDevice is a physical volume where docker data would be stored
	dockerDevice string `validate:"required"`
}

// LoadConfig loads essential parameters from YAML
func LoadConfig(t *testing.T, configBytes []byte, cfg *ProvisionerConfig) {
	err := yaml.Unmarshal(configBytes, cfg)
	require.NoError(t, err, string(configBytes))

	switch cfg.CloudProvider {
	case "azure":
		require.NotNil(t, cfg.Azure)
		cfg.dockerDevice = cfg.Azure.DockerDevice
	case "aws":
		require.NotNil(t, cfg.AWS)
		cfg.dockerDevice = cfg.AWS.DockerDevice
	default:
		t.Fatal("unknown cloud provider %s", cfg.CloudProvider)
	}
}

// Tag returns current tag of a config
func (config ProvisionerConfig) Tag() string {
	return config.tag
}

// WithTag returns copy of config applying extended tag to it
func (config ProvisionerConfig) WithTag(tag string) ProvisionerConfig {
	cfg := config
	if cfg.tag == "" {
		cfg.tag = tag
	} else {
		cfg.tag = fmt.Sprintf("%s-%s", cfg.tag, tag)
	}
	cfg.StateDir = filepath.Join(cfg.StateDir, tag)

	return cfg
}

// WithNodes returns copy of config with specific number of nodes
func (config ProvisionerConfig) WithNodes(nodes uint) ProvisionerConfig {
	extra := fmt.Sprintf("%dn", nodes)

	cfg := config
	cfg.nodeCount = nodes
	cfg.tag = fmt.Sprintf("%s-%s", cfg.tag, extra)
	cfg.StateDir = filepath.Join(cfg.StateDir, extra)

	return cfg
}

// WithOS returns copy of config with specific OS
func (config ProvisionerConfig) WithOS(os string) ProvisionerConfig {
	cfg := config
	cfg.os = os
	cfg.tag = fmt.Sprintf("%s-%s", cfg.tag, os)
	cfg.StateDir = filepath.Join(cfg.StateDir, os)

	return cfg
}

// WithStorageDriver returns copy of config with specific storage driver
func (config ProvisionerConfig) WithStorageDriver(storageDriver string) ProvisionerConfig {
	cfg := config
	if storageDriver == "loopback" {
		cfg.storageDriver = "devicemapper"
		cfg.dockerDevice = ""
	} else {
		cfg.storageDriver = storageDriver
	}
	cfg.tag = fmt.Sprintf("%s-%s", cfg.tag, storageDriver)
	cfg.StateDir = filepath.Join(cfg.StateDir, storageDriver)

	return cfg
}

// validateConfig checks that key parameters are present
func validateConfig(t *testing.T, config ProvisionerConfig) {
	err := validator.New().Struct(&config)
	if err == nil {
		return
	}

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		t.Errorf("Errors in config: %+v", config)
		for _, fieldError := range validationErrors {
			t.Errorf(" * %s=\"%v\" fails \"%s\"", fieldError.Field(), fieldError.Value(), fieldError.Tag())
		}
		require.FailNow(t, "Fix ProvisionerConfig")
	}
}
