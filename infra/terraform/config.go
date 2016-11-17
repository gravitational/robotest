package terraform

import (
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/trace"
)

// Validate validates the configuration
func (r *Config) Validate() error {
	var errors []error
	if r.AccessKey == "" {
		errors = append(errors, trace.BadParameter("access key is required"))
	}
	if r.SecretKey == "" {
		errors = append(errors, trace.BadParameter("secret key is required"))
	}
	if r.SSHKeyPath == "" {
		errors = append(errors, trace.BadParameter("SSH key path is required"))
	}
	if r.InstallerURL == "" {
		errors = append(errors, trace.BadParameter("installer URL is required"))
	}
	if r.ScriptPath == "" {
		errors = append(errors, trace.BadParameter("script path is required"))
	}
	if r.InstallNodes > r.Nodes {
		errors = append(errors, trace.BadParameter("cannot use more nodes than the capacity for installation (%v > %v)",
			r.InstallNodes, r.Nodes))
	}
	if r.InstallNodes == 0 {
		r.InstallNodes = r.Nodes
	}
	return trace.NewAggregate(errors...)
}

type Config struct {
	infra.Config
	// AccessKey is AWS access key
	AccessKey string `json:"access_key" env:"ROBO_ACCESS_KEY"`
	// SecretKey is AWS secret key
	SecretKey string `json:"secret_key" env:"ROBO_SECRET_KEY"`
	// Region is AWS region to deploy to
	Region string `json:"region" env:"ROBO_REGION"`
	// KeyPair is AWS key pair to use
	KeyPair string `json:"key_pair" env:"ROBO_KEY_PAIR"`
	// SSHKeyPath is the path to the private SSH key to connect to provisioned machines
	SSHKeyPath string `json:"ssh_key_path" env:"ROBO_SSH_KEY_PATH"`
	// InstanceType is AWS instance type
	InstanceType string `json:"instance_type" env:"ROBO_INSTANCE_TYPE"`
	// ScriptPath is the path to the terraform script for provisioning
	ScriptPath string `json:"script_path" env:"ROBO_SCRIPT_PATH"`
	// Nodes defines the capacity of the cluster to provision
	Nodes int `json:"nodes" env:"ROBO_NODES"`
	// InstallNodes defines the number of nodes to use for installation (must be <= Nodes).
	// For expand/shrink operations, InstallNodes can be Nodes-1 to allow the cluster to grow
	// TODO: replace Nodes/InstallNodes with dynamic expansion
	InstallNodes int `json:"install_nodes" env:"ROBO_INSTALL_NODES"`
	// InstallerURL is AWS S3 URL with the installer
	InstallerURL string `json:"installer_url" env:"ROBO_INSTALLER_URL"`
}
