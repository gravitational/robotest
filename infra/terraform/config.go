package terraform

import (
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/trace"
)

// Validate validates the configuration
func (c *Config) Validate() error {
	if (c.ProvisionTo == "aws" && c.AWS != nil && c.AWS.SSHUser != "" && c.AWS.SSHKeyPath != "") ||
		(c.ProvisionTo == "azure" && c.Azure != nil && c.Azure.SSHUser != "" && c.Azure.SSHKeyPath != "") {
		return nil
	}

	return trace.Errorf("Missing or invalid configuration for cloud %s", c.ProvisionTo)
}

func (c Config) sshConfig() (user, keypath string) {
	switch c.ProvisionTo {
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
	ProvisionTo string `validate:"required,eq=aws|eq=azure"`
	// AWS defines AWS connection parameters
	AWS *infra.AWSConfig
	// Azure defines Azure connection parameters
	Azure *infra.AzureConfig

	// ScriptPath is the path to the terraform script or directory for provisioning
	ScriptPath string `json:"script_path" env:"ROBO_SCRIPT_PATH" validate:"required"`
	// NumNodes defines the capacity of the cluster to provision
	NumNodes int `json:"nodes" env:"ROBO_NUM_NODES" validate:"gte=1"`
	// InstallerURL is AWS S3 URL with the installer
	InstallerURL string `json:"installer_url" env:"ROBO_INSTALLER_URL" validate:"required,url`
}
