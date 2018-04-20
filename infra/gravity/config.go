package gravity

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/providers/aws"
	"github.com/gravitational/robotest/infra/providers/azure"
	"github.com/gravitational/robotest/infra/providers/gce"
	"github.com/gravitational/robotest/infra/providers/ops"
	"github.com/gravitational/robotest/lib/constants"

	"github.com/go-yaml/yaml"
	"github.com/gravitational/trace"
	"github.com/stretchr/testify/require"
	"gopkg.in/go-playground/validator.v9"
)

// OS represents OS vendor/version
type OS struct {
	Vendor, Version string
}

// UnmarshalText interprets OS from a textual representation
func (os *OS) UnmarshalText(b []byte) error {
	str, err := strconv.Unquote(string(b))
	if err != nil {
		return trace.ConvertSystemError(err)
	}
	split := strings.Split(str, ":")
	if len(split) != 2 {
		return trace.BadParameter("OS should be in format vendor:version, got %q", b)
	}
	os.Vendor = split[0]
	os.Version = split[1]
	return nil
}

// Strings returns a textual representation of this OS object
func (os OS) String() string {
	return fmt.Sprintf("%s:%s", os.Vendor, os.Version)
}

type StorageDriver string

func (drv *StorageDriver) UnmarshalText(b []byte) error {
	name, err := strconv.Unquote(string(b))
	if err != nil {
		return trace.ConvertSystemError(err)
	}
	switch name {
	case constants.DeviceMapper, constants.Overlay, constants.Overlay2, constants.Loopback, constants.ManifestStorageDriver:
		*drv = StorageDriver(name)
		return nil
	default:
		return trace.BadParameter("unknown storage driver %s", name)
	}
}

// Driver validates and returns driver name
func (drv StorageDriver) Driver() string {
	return string(drv)
}

// ProvisionerConfig defines parameters required to provision hosts
// CloudProvider, AWS, Azure, ScriptPath and InstallerURL
type ProvisionerConfig struct {
	// DeployTo defines cloud to deploy to
	CloudProvider string `yaml:"cloud" validate:"required,eq=aws|eq=azure|eq=ops"`
	// AWS defines AWS connection parameters
	AWS *aws.Config `yaml:"aws"`
	// Azure defines Azure connection parameters
	Azure *azure.Config `yaml:"azure"`
	// GCE defines Google Compute Engine connection parameters
	GCE *gce.Config `yaml:"azure"`
	// Ops defines Ops Center connection parameters
	Ops *ops.Config `yaml:"ops"`

	// ScriptPath is the path to the terraform script or directory for provisioning
	ScriptPath string `yaml:"script_path" validate:"required"`
	// InstallerURL is AWS S3 URL with the installer
	InstallerURL string `yaml:"installer_url" validate:"required,url`
	// StateDir defines base directory where to keep state (i.e. terraform configs/vars)
	StateDir string `yaml:"state_dir" validate:"required"`

	// Tag will group provisioned resources under for easy removal afterwards
	tag string `validate:"required"`
	// NodeCount defines amount of nodes to be provisioned
	NodeCount uint `validate:"gte=1"`
	// OS defines one of supported operating systems
	os OS `validate:"required"`
	// storageDriver specifies the storage driver for Docker
	storageDriver StorageDriver
	// dockerDevice is a physical volume where Docker data would be stored
	dockerDevice string `validate:"required"`
	// clusterName is the name of the resulting robotest cluster
	clusterName string
}

// LoadConfig loads essential parameters from YAML
func LoadConfig(t *testing.T, configBytes []byte, cfg *ProvisionerConfig) {
	err := yaml.Unmarshal(configBytes, cfg)
	require.NoError(t, err, string(configBytes))

	switch cfg.CloudProvider {
	case "azure":
		require.NotNil(t, cfg.Azure)
		cfg.dockerDevice = cfg.Azure.DockerDevice
		azureRegions = NewCloudRegions(strings.Split(cfg.Azure.Location, ","))
	case "aws":
		require.NotNil(t, cfg.AWS)
		cfg.dockerDevice = cfg.AWS.DockerDevice
	case "ops":
		require.NotNil(t, cfg.Ops)
		// set AWS environment variables to be used by subsequent commands
		os.Setenv("AWS_ACCESS_KEY_ID", cfg.Ops.EC2AccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", cfg.Ops.EC2SecretKey)
		// normally the docker device is set to /dev/abc before gravity is installed
		// for throughput testing. However, when using the ops center for provisioning
		// the raw block device will have a partition on it, so we want to instead test
		// on the installation directory
		cfg.dockerDevice = "/var/lib/gravity"
	default:
		t.Fatalf("unknown cloud provider %s", cfg.CloudProvider)
	}
}

// Tag returns the configured tag.
// Tag is a unique robotest cluster identifier
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
	cfg.NodeCount = nodes
	cfg.tag = fmt.Sprintf("%s-%s", cfg.tag, extra)
	cfg.StateDir = filepath.Join(cfg.StateDir, extra)

	return cfg
}

// WithOS returns copy of config with specific OS
func (config ProvisionerConfig) WithOS(os OS) ProvisionerConfig {
	cfg := config
	cfg.os = os
	cfg.tag = fmt.Sprintf("%s-%s%s", cfg.tag, os.Vendor, os.Version)
	cfg.StateDir = filepath.Join(cfg.StateDir, fmt.Sprintf("%s%s", os.Vendor, os.Version))

	return cfg
}

// WithStorageDriver returns copy of config with specific storage driver
func (config ProvisionerConfig) WithStorageDriver(storageDriver StorageDriver) ProvisionerConfig {
	cfg := config
	cfg.storageDriver = storageDriver

	tag := storageDriver.Driver()
	if tag == "" {
		tag = "none"
	}
	cfg.tag = fmt.Sprintf("%s-%s", cfg.tag, tag)
	cfg.StateDir = filepath.Join(cfg.StateDir, tag)

	return cfg
}

// validateConfig checks that key parameters are present
func validateConfig(config ProvisionerConfig) error {
	switch config.CloudProvider {
	case constants.AWS, constants.Azure, constants.Ops:
	default:
		return trace.BadParameter("unknown cloud provider %s", config.CloudProvider)
	}

	err := validator.New().Struct(&config)
	if err == nil {
		return nil
	}

	var errs []error
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			errs = append(errs,
				trace.BadParameter(` * %s="%v" fails "%s"`,
					fieldError.Field(), fieldError.Value(), fieldError.Tag()))
		}
	}
	return trace.NewAggregate(errs...)
}

// CloudRegions is used for round-robin distribution of workload across regions
type CloudRegions struct {
	sync.Mutex
	idx     int
	regions []string
}

func NewCloudRegions(regions []string) *CloudRegions {
	out := make([]string, len(regions))
	perm := rand.Perm(len(regions))
	for i, v := range perm {
		out[v] = regions[i]
	}

	return &CloudRegions{idx: 0, regions: regions}
}

func (r *CloudRegions) Next() (region string) {
	r.Lock()
	defer r.Unlock()

	r.idx = (r.idx + 1) % len(r.regions)
	return r.regions[r.idx]
}

var cloudRegions *CloudRegions
