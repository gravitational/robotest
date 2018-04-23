package gce

// Config specifies Google Compute Engine specific parameters
type Config struct {
	// Project name
	// https://cloud.google.com/resource-manager/docs/creating-managing-projects
	Project string `json:"project" yaml:"project"`
	// Credentials names the service account file
	// https://cloud.google.com/compute/docs/api/how-tos/authorization
	Credentials string `json:"credentials" yaml:"credentials" validate:"required"`
	// Region specifies the datacenter region to install into.
	// Can be a comma-separated list of regions.
	// https://cloud.google.com/compute/docs/regions-zones/
	Region string `json:"region" yaml:"region"`
	// Zone specifies the datacenter zone to install into.
	// It is the required parameter as it defines the region as well.
	// https://cloud.google.com/compute/docs/regions-zones/
	Zone string `json:"zone" yaml:"zone" validate:"required"`
	// VMType specifies the type of VP to provision
	// https://cloud.google.com/compute/docs/machine-types
	VMType string `json:"vm_type" yaml:"vm_type" validate:"required"`
	// SSHKeyPath specifies the location of the SSH private key for remote access
	SSHKeyPath string `json:"-" yaml:"key_path" validate:"required"`
	// SSHUser defines SSH user to connect to the provisioned machines
	SSHUser string `json:"ssh_user" yaml:"ssh_user" validate:"required"`
	// DockerDevice specifies the block device for Docker
	DockerDevice string `json:"docker_device" yaml:"docker_device"`
}
