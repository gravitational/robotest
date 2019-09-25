package gce

// Config specifies Google Compute Engine specific parameters
type Config struct {
	// Project name
	// https://cloud.google.com/resource-manager/docs/creating-managing-projects
	Project string `json:"project,omitempty" yaml:"project"`
	// Credentials names the service account file
	// https://cloud.google.com/compute/docs/api/how-tos/authorization
	Credentials string `json:"credentials" yaml:"credentials" validate:"required"`
	// Region specifies the datacenter region to install into.
	// Can be a comma-separated list of regions.
	// https://cloud.google.com/compute/docs/regions-zones/
	Region string `json:"region,omitempty" yaml:"region"`
	// Zone specifies the datacenter zone to install into.
	// It is the required parameter as it defines the region as well.
	// https://cloud.google.com/compute/docs/regions-zones/
	Zone string `json:"zone,omitempty" yaml:"zone"`
	// VMType specifies the type of machine to provision
	// https://cloud.google.com/compute/docs/machine-types
	VMType string `json:"vm_type" yaml:"vm_type" validate:"required"`
	// SSHUser defines SSH user to connect to the provisioned machines.
	// Required attribute.
	// Will be determined based on selected cloud provder.
	SSHUser string `json:"os_user" yaml:"os_user"`
	// SSHPublicKeyPath specifies the location of the public SSH key
	SSHPublicKeyPath string `json:"ssh_pub_key_path" yaml:"ssh_pub_key_path" validate:"required"`
	// NodeTag specifies the node tag to use on GCE.
	// Required attribute.
	// Will be computed based on the cluster name during provisioning
	NodeTag string `json:"node_tag" yaml:"node_tag"`

	// SSHKeyPath specifies the location of the SSH private key for remote access
	SSHKeyPath string `json:"-" yaml:"ssh_key_path" validate:"required"`
	// VarFilePath is the path to file with custom terraform variables
	VarFilePath string `json:"-" yaml:"var_file_path"`
}
