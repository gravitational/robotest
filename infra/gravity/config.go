package gravity

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gravitational/robotest/infra"

	"github.com/go-yaml/yaml"
	"github.com/stretchr/testify/require"
	"gopkg.in/go-playground/validator.v9"
)

// ProvisionerConfig defines parameters required to provision hosts
// CloudProvider, AWS, Azure, ScriptPath and InstallerURL
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

	// Tag will group provisioned resources under for easy removal afterwards
	tag string `validate:"required"`
	// StateDir defines base directory where to keep state (i.e. terraform configs/vars)
	stateDir string `validate:"required"`
	// NodeCount defines amount of nodes to be provisioned
	nodeCount uint `validate:"gte=1"`
	// OS defines one of supported operating systems
	os string `validate:"required,eq=ubuntu|eq=debian|eq=rhel|eq=centos"`
	// dockerStorageDriver defines Docker storage driver
	storageDriver string `validate:"required,eq=overlay2|devicemapper"`
}

// LoadConfig loads essential parameters from YAML
func LoadConfig(t *testing.T, configFile, stateDir, tag string) *ProvisionerConfig {
	require.NotEmpty(t, configFile, "config file")
	f, err := os.Open(configFile)
	require.NoError(t, err, configFile)
	defer f.Close()

	configBytes, err := ioutil.ReadAll(f)
	require.NoError(t, err)

	cfg := ProvisionerConfig{}
	err = yaml.Unmarshal(configBytes, &cfg)
	require.NoError(t, err, configFile)

	if cfg.tag = tag; cfg.tag == "" {
		now := time.Now()
		cfg.tag = fmt.Sprintf("RT-%02d%02d-%02d%02d",
			now.Month(), now.Day(), now.Hour(), now.Minute())
	}

	if cfg.stateDir = stateDir; cfg.stateDir == "" {
		cfg.stateDir, err = ioutil.TempDir("", "robotest")
		require.NoError(t, err, "tmp dir")
	}

	cfg.stateDir = filepath.Join(cfg.stateDir, cfg.tag)

	return &cfg
}

// Tag returns current tag of a config
func (config *ProvisionerConfig) Tag() string {
	return config.tag
}

// WithTag returns copy of config applying extended tag to it
func (config *ProvisionerConfig) WithTag(tag string) *ProvisionerConfig {
	cfg := *config
	cfg.tag = fmt.Sprintf("%s-%s", cfg.tag, tag)
	cfg.stateDir = filepath.Join(cfg.stateDir, tag)

	return &cfg
}

// WithNodes returns copy of config with specific number of nodes
func (config *ProvisionerConfig) WithNodes(nodes uint) *ProvisionerConfig {
	extra := fmt.Sprintf("%d_nodes", nodes)

	cfg := *config
	cfg.nodeCount = nodes
	cfg.tag = fmt.Sprintf("%s-%s", cfg.tag, extra)
	cfg.stateDir = filepath.Join(cfg.stateDir, extra)

	return &cfg
}

// WithOS returns copy of config with specific OS
func (config *ProvisionerConfig) WithOS(os string) *ProvisionerConfig {
	cfg := *config
	cfg.os = os
	cfg.tag = fmt.Sprintf("%s-%s", cfg.tag, os)
	cfg.stateDir = filepath.Join(cfg.stateDir, os)

	return &cfg
}

// WithStorageDriver returns copy of config with specific storage driver
func (config *ProvisionerConfig) WithStorageDriver(storageDriver string) *ProvisionerConfig {
	cfg := *config
	cfg.storageDriver = storageDriver
	cfg.tag = fmt.Sprintf("%s-%s", cfg.tag, storageDriver)
	cfg.stateDir = filepath.Join(cfg.stateDir, storageDriver)

	return &cfg
}

// validateConfig checks that key parameters are present
func validateConfig(t *testing.T, config *ProvisionerConfig) {
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
