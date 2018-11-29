package gravity

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

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

// UnmarshalText interprets b as an OS vendor with a version.
// I.e. given:
//
//   "vendor:version", it populates this OS instance accordingly
func (os *OS) UnmarshalText(b []byte) error {
	split := bytes.Split(b, []byte(":"))
	if len(split) != 2 {
		return trace.BadParameter("OS should be in format vendor:version, got %q", b)
	}
	os.Vendor = string(split[0])
	os.Version = string(split[1])
	return nil
}

// String returns a textual representation of this OS instance
func (os OS) String() string {
	return fmt.Sprintf("%s:%s", os.Vendor, os.Version)
}

// StorageDriver specifies a Docker storage driver by name
type StorageDriver string

// UnmarshalText interprets b as a Docker storage driver name
func (drv *StorageDriver) UnmarshalText(name []byte) error {
	switch string(name) {
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
	CloudProvider string `yaml:"cloud" validate:"required,eq=aws|eq=azure|eq=gce|eq=ops"`
	// AWS defines AWS connection parameters
	AWS *aws.Config `yaml:"aws"`
	// Azure defines Azure connection parameters
	Azure *azure.Config `yaml:"azure"`
	// GCE defines Google Compute Engine connection parameters
	GCE *gce.Config `yaml:"gce"`
	// Ops defines Ops Center connection parameters
	Ops *ops.Config `yaml:"ops"`

	// ScriptPath is the path to the terraform script or directory for provisioning
	ScriptPath string `yaml:"script_path" validate:"required"`
	// InstallerURL specifies the location of the installer tarball.
	// Can either be a local path or S3 URL
	InstallerURL string `yaml:"installer_url" validate:"required"`
	// GravityURL specifies the location of the up-to-date gravity binary.
	// Can either be a local path or S3 URL
	GravityURL string `yaml:"gravity_url" validate:"required"`
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
	clusterName  string
	cloudRegions *cloudRegions
}

// LoadConfig loads essential parameters from YAML
func LoadConfig(t *testing.T, configBytes []byte) (cfg ProvisionerConfig) {
	err := yaml.Unmarshal(configBytes, &cfg)
	require.NoError(t, err, string(configBytes))

	switch cfg.CloudProvider {
	case constants.Azure:
		require.NotNil(t, cfg.Azure)
		cfg.dockerDevice = cfg.Azure.DockerDevice
		cfg.cloudRegions = newCloudRegions(strings.Split(cfg.Azure.Location, ","))
	case constants.AWS:
		require.NotNil(t, cfg.AWS)
		cfg.dockerDevice = cfg.AWS.DockerDevice
	case constants.GCE:
		require.NotNil(t, cfg.GCE)
		cfg.dockerDevice = cfg.GCE.DockerDevice
		cfg.cloudRegions = newCloudRegions(strings.Split(cfg.GCE.Region, ","))
	case constants.Ops:
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
	return cfg
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
	case constants.AWS, constants.Azure, constants.GCE, constants.Ops:
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

// newCloudRegions returns a new list of cloud regions in
// random order
func newCloudRegions(regions []string) *cloudRegions {
	out := make([]string, len(regions))
	perm := rand.Perm(len(regions))
	for i, v := range perm {
		out[v] = regions[i]
	}

	return &cloudRegions{idx: 0, regions: regions}
}

// Next returns the next region.
// It wraps around once it has reached the end of the list
func (r *cloudRegions) Next() (region string) {
	r.Lock()
	defer r.Unlock()

	r.idx = (r.idx + 1) % len(r.regions)
	return r.regions[r.idx]
}

// cloudRegions is used for round-robin distribution of workload across regions
type cloudRegions struct {
	sync.Mutex
	idx     int
	regions []string
}
