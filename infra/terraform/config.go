package terraform

import (
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/trace"
)

// Validate validates the configuration
func (c *Config) Validate() error {
	if (c.CloudProvider == "aws" && c.AWS != nil && c.AWS.SSHUser != "" && c.AWS.SSHKeyPath != "") ||
		(c.CloudProvider == "azure" && c.Azure != nil && c.Azure.SSHUser != "" && c.Azure.SSHKeyPath != "") {
		return nil
	}

	return trace.Errorf("Missing or invalid configuration for cloud %s", c.CloudProvider)
}

func (c Config) SSHConfig() (user, keypath string) {
	switch c.CloudProvider {
	case "aws":
		return c.AWS.SSHUser, c.AWS.SSHKeyPath
	case "azure":
		return c.Azure.SSHUser, c.Azure.SSHKeyPath
	default:
		return "", ""
	}
}

type Config struct {
	infra.Config

	// DeployTo defines cloud to deploy to
	CloudProvider string `validate:"required,eq=aws|eq=azure"`
	// AWS defines AWS connection parameters
	AWS *infra.AWSConfig
	// Azure defines Azure connection parameters
	Azure *infra.AzureConfig
	// OS defines OS flavor, ubuntu | redhat | centos | debian
	OS string `json:"os" yaml:"os" validate:"required,eq=ubuntu|eq=redhat|eq=centos|eq=debian"`

	// ScriptPath is the path to the terraform script or directory for provisioning
	ScriptPath string `json:"script_path" validate:"required"`
	// NumNodes defines the capacity of the cluster to provision
	NumNodes int `json:"nodes" validate:"gte=1"`
	// InstallerURL is AWS S3 URL with the installer
	InstallerURL string `json:"installer_url" validate:"required,url`
}
