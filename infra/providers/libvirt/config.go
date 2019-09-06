package libvirt

// Config specifies libvirt specific parameters
type Config struct {
	// SSHKeyPath specifies the location of the SSH private key for remote access
	SSHKeyPath string `json:"-" yaml:"ssh_key_path" validate:"required"`
	// SSHUser defines SSH user to connect to the provisioned machines.
	SSHUser string `json:"ssh_user" yaml:"ssh_user" validate:"required"`
	// SSHPublicKeyPath specifies the location of the public SSH key
	SSHPublicKeyPath string `json:"ssh_pub_key_path" yaml:"ssh_pub_key_path" validate:"required"`
	// DockerDevice specifies the block device for Docker
	DockerDevice string `json:"docker_device,omitempty" yaml:"docker_device"`
	// CPU specifies the number of vCPUs per machine
	CPU int `json:"cpu" yaml:"cpu" validate:"required"`
	// Memory specifies the amount of memory per machine in MB
	Memory int `json:"memory" yaml:"memory" validate:"required"`
}
