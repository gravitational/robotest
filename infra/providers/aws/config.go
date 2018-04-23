package aws

// Config describes AWS EC2 test configuration
type Config struct {
	// AccessKey http://docs.aws.amazon.com/general/latest/gr/managing-aws-access-keys.html
	AccessKey string `json:"access_key" yaml:"access_key" validate:"required"`
	// SecretKey http://docs.aws.amazon.com/general/latest/gr/managing-aws-access-keys.html
	SecretKey string `json:"secret_key" yaml:"secret_key" validate:"required"`
	// Region specifies the EC2 region to install into
	Region string `json:"region" yaml:"region" validate:"required"`
	// KeyPair specifies the name of the SSH key pair to use for provisioning
	// nodes
	KeyPair string `json:"key_pair" yaml:"key_pair" validate:"required"`
	// VPC defines the Amazon VPC to install into.
	// Specify "Create new" to create a new VPC for this test run
	VPC string `json:"vpc" yaml:"vpc" validate:"required"`
	// SSHKeyPath specifies the location of the SSH key to use for remote access.
	// Mandatory only with terraform provisioner
	SSHKeyPath string `json:"key_path" yaml:"key_path"`
	// SSHUser defines SSH user used to connect to the provisioned machines
	SSHUser string `json:"ssh_user" yaml:"ssh_user" validate:"required"`
	// InstanceType defines the type of AWS EC2 instance to boot.
	// Relevant only with terraform provisioner.
	// Defaults are specific to the terraform script used (if any)
	InstanceType string `json:"instance_type,omitempty" yaml:"instance_type"`
	// ExpandProfile specifies an optional name of the server profile for AWS expand operation.
	// If the profile is unspecified, the test will use the first available.
	ExpandProfile string `json:"expand_profile" yaml:"expand_profile"`
	// ExpandAwsInstanceType specifies an optional instance type for AWS expand operation
	ExpandAWSInstanceType string `json:"expand_instance_type" yaml:"expand_instance_type"`
	// ClusterName defines tagging and placement group for resources allocated
	ClusterName string `json:"cluster_name" yaml:"cluster_name"`
	// DockerDevice block device for docker data - set to /dev/xvdb
	DockerDevice string `json:"docker_device" yaml:"docker_device" validate:"required"`
}

// FIXME: replace with embedded validation rules
func (r Config) IsEmpty() bool {
	return r.AccessKey == "" && r.SecretKey == ""
}
