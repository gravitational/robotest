package terraform

import (
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/providers/aws"
	"github.com/gravitational/robotest/infra/providers/azure"
	"github.com/gravitational/robotest/infra/providers/gce"
	"github.com/gravitational/robotest/lib/constants"

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

	if len(errors) != 0 {
		return trace.NewAggregate(errors...)
	}

	switch c.CloudProvider {
	case constants.AWS:
		if c.AWS == nil {
			return trace.BadParameter("AWS configuration is required")
		}
		if c.AWS.SSHUser == "" || c.AWS.SSHKeyPath == "" {
			return trace.BadParameter("AWS SSH access configuration is required")
		}
	case constants.Azure:
		if c.Azure == nil {
			return trace.BadParameter("Azure configuration is required")
		}
		if c.Azure.SSHUser == "" || c.Azure.SSHKeyPath == "" {
			return trace.BadParameter("Azure SSH access configuration is required")
		}
	case constants.GCE:
		if c.GCE == nil {
			return trace.BadParameter("GCE configuration is required")
		}
		if c.GCE.SSHUser == "" || c.GCE.SSHKeyPath == "" {
			return trace.BadParameter("GCE SSH access configuration is required")
		}
	}

	return nil
}

func (c Config) SSHConfig() (user, keypath string) {
	switch c.CloudProvider {
	case constants.AWS:
		return c.AWS.SSHUser, c.AWS.SSHKeyPath
	case constants.Azure:
		return c.Azure.SSHUser, c.Azure.SSHKeyPath
	case constants.GCE:
		return c.GCE.SSHUser, c.GCE.SSHKeyPath
	default:
		return "", ""
	}
}

// Config represents terraform provisioning configuration
type Config struct {
	// Config specifies common infrastructure configuration
	infra.Config
	// CloudProvider defines cloud to deploy to
	CloudProvider string `validate:"required,eq=aws|eq=azure|eq=gce"`
	// AWS defines AWS connection parameters
	AWS *aws.Config
	// Azure defines Azure connection parameters
	Azure *azure.Config
	// GCE defines Google Compute Engine connection parameters
	GCE *gce.Config
	// OS specified the OS distribution
	OS string `json:"os" yaml:"os" validate:"required,eq=ubuntu|eq=redhat|eq=centos|eq=debian|eq=suse"`
	// ScriptPath is the path to the terraform script or directory for provisioning
	ScriptPath string `json:"script_path" validate:"required"`
	// NumNodes defines the capacity of the cluster to provision
	NumNodes int `json:"nodes" validate:"gte=1"`
	// InstallerURL is AWS S3 URL with the installer
	InstallerURL string `json:"installer_url" validate:"required,url"`
	// DockerDevice block device for docker data - set to /dev/xvdb
	DockerDevice string `json:"docker_device" yaml:"docker_device" validate:"required"`
	// PostInstallerScript defines a path to the script on a remote node
	// that is executed after the installer has been downloaded
	PostInstallerScript string `json:"post_installer_script" yaml:"post_installer_script"`
	// VariablesFile is the file with custom terraform variables
	VariablesFile string `json:"custom_vars_file" yaml:"variables_file"`
	// OnpremProvider specifies usage of onprem provider for installation
	OnpremProvider bool `json:"onprem_provider" yaml:"onprem_provider"`
	// Preemptible is whether to use preemptible VMs
	Preemptible bool `json:"preemptible" yaml:"preemptible"`
}
