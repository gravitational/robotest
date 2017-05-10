package infra

import "github.com/gravitational/trace"

func (r *Config) Validate() error {
	if r.ClusterName == "" {
		return trace.BadParameter("cluster name is required")
	}
	return nil
}

type Config struct {
	// ClusterName is the name assigned to the provisioned machines
	ClusterName string `json:"cluster_name" env:"ROBO_CLUSTER_NAME"`
}

// ProvisionerState defines the state configuration for a cluster
// provisioned with a specific provisioner
type ProvisionerState struct {
	// Dir defines the location where provisioner stores state
	Dir string `json:"state_dir"`
	// InstallerAddr is the address of the installer node
	InstallerAddr string `json:"installer_addr,omitempty"`
	// Nodes is a list of all nodes in the cluster
	Nodes []StateNode `json:"nodes"`
	// Allocated defines the allocated subset
	Allocated []string `json:"allocated_nodes"`
}

// StateNode describes a single cluster node
type StateNode struct {
	// Addr is the address of this node
	Addr string `json:"addr"`
	// KeyPath defines the location of the SSH key
	KeyPath string `json:"key_path"`
}

// AWSConfig describes AWS EC2 test configuration
type AWSConfig struct {
	AccessKey string `json:"access_key" yaml:"access_key" env:"ROBO_AWS_ACCESS_KEY" validate:"required"`
	SecretKey string `json:"secret_key" yaml:"secret_key" env:"ROBO_AWS_SECRET_KEY" validate:"required"`
	// Region specifies the EC2 region to install into
	Region string `json:"region" yaml:"region" env:"ROBO_AWS_REGION" validate:"required"`
	// KeyPair specifies the name of the SSH key pair to use for provisioning
	// nodes
	KeyPair string `json:"key_pair" yaml:"key_pair" env:"ROBO_AWS_KEY_PAIR" validate:"required"`
	// VPC defines the Amazon VPC to install into.
	// Specify "Create new" to create a new VPC for this test run
	VPC string `json:"vpc" yaml:"vpc" env:"ROBO_AWS_VPC" validate:"required"`
	// KeyPath specifies the location of the SSH key to use for remote access.
	// Mandatory only with terraform provisioner
	SSHKeyPath string `json:"key_path" yaml:"key_path" env:"ROBO_AWS_KEY_PATH"`
	// SSHUser defines SSH user used to connect to the provisioned machines
	SSHUser string `json:"ssh_user" yaml:"ssh_user" env:"ROBO_AWS_SSH_USER" validate:"required"`
	// InstanceType defines the type of AWS EC2 instance to boot.
	// Relevant only with terraform provisioner.
	// Defaults are specific to the terraform script used (if any)
	InstanceType string `json:"omitempty,instance_type" yaml:"instance_type" env:"ROBO_AWS_INSTANCE_TYPE"`
	// ExpandProfile specifies an optional name of the server profile for AWS expand operation.
	// If the profile is unspecified, the test will use the first available.
	ExpandProfile string `json:"expand_profile" yaml:"expand_profile" env:"ROBO_AWS_EXPAND_PROFILE"`
	// ExpandAwsInstanceType specifies an optional instance type for AWS expand operation
	ExpandAWSInstanceType string `json:"expand_instance_type" yaml:"expand_instance_type" env:"ROBO_AWS_EXPAND_INSTANCE_TYPE"`
}

// FIXME : replace with embedded validation rules
func (r *AWSConfig) IsEmpty() bool {
	return r.AccessKey == "" && r.SecretKey == ""
}

// AzureConfig specifies Azure cloud specific parameters
// see https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal
// to obtain subscription_id, client_id, client_secret and tenant_id
type AzureConfig struct {
	SubscriptionId string `json:"subscription_id" yaml:"subscription_id" validate:"required"`
	ClientId       string `json:"client_id" yaml:"client_id" validate:"required"`
	ClientSecret   string `json:"client_secret" yaml:"client_secret" validate:"required"`
	TenantId       string `json:"tenant_id" yaml:"tenant_id" validate:"required"`

	// Resource Group defines logical grouping of resources, and makes it easy to wipe them once not needed
	ResourceGroup string `json:"azure_resource_group" yaml:"resource_group" validate:"required"`
	// Location specifies the datacenter region to install into
	// https://azure.microsoft.com/en-ca/regions/
	Location string `json:"location" yaml:"location" validate:"required"`
	// VM instance type
	// https://docs.microsoft.com/en-us/cli/azure/vm#list-sizes
	VmType string `json:"vm_type" yaml:"vm_type" validate:"required"`
	// KeyPath specifies the location of the SSH private key to use for remote access
	SSHKeyPath string `json:"-" yaml:"key_path" validate:"required"`
	// AuthorizedKeysPath specifies ssh/authorized_keys file to be placed on remote machine
	AuthorizedKeysPath string `json:"ssh_authorized_keys_path" yaml:"authorized_keys_path" validate:"required"`
	// SSHUser defines SSH user used to connect to the provisioned machines
	SSHUser string `json:"ssh_user" yaml:"ssh_user" validate:"required"`
}
