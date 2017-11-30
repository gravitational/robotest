package terraform

import (
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/trace"

	"gopkg.in/go-playground/validator.v9"
)

// Validate validates the configuration
func (c *Config) Validate() error {
	errors := []error{}

	err := validator.New().Struct(c)
	if validationErrors, ok := err.(validator.ValidationErrors); err != nil && ok {
		for _, fieldError := range validationErrors {
			errors = append(errors,
				trace.Errorf(" * %s=\"%v\" fails \"%s\"", fieldError.Field(), fieldError.Value(), fieldError.Tag()))
		}
	}

	hasValidAWSSSH := c.CloudProvider == "aws" && c.AWS != nil && c.AWS.SSHUser != "" && c.AWS.SSHKeyPath != ""
	hasValidAzureSSH := c.CloudProvider == "azure" && c.Azure != nil && c.Azure.SSHUser != "" && c.Azure.SSHKeyPath != ""

	if (hasValidAWSSSH || hasValidAzureSSH) == false {
		errors = append(errors,
			trace.Errorf("SSH configuration missing for %s", c.CloudProvider))
	}

	return trace.NewAggregate(errors...)
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
	InstallerURL string `json:"installer_url" validate:"required,url"`
	// DockerDevice block device for docker data - set to /dev/xvdb
	DockerDevice string `json:"docker_device" yaml:"docker_device" validate:"required"`
}
