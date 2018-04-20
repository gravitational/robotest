package azure

// Config specifies Azure Cloud specific parameters
type Config struct {
	// SubscriptionId https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal
	SubscriptionId string `json:"subscription_id" yaml:"subscription_id" validate:"required"`
	// ClientId https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal
	ClientId string `json:"client_id" yaml:"client_id" validate:"required"`
	// ClientSecret https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal
	ClientSecret string `json:"client_secret" yaml:"client_secret" validate:"required"`
	// TenantId https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal
	TenantId string `json:"tenant_id" yaml:"tenant_id" validate:"required"`
	// Resource Group defines logical grouping of resources, and makes it easy to wipe them once not needed
	ResourceGroup string `json:"azure_resource_group" yaml:"resource_group"`
	// Location specifies the datacenter region to install into
	// https://azure.microsoft.com/en-ca/regions/
	Location string `json:"location" yaml:"location" validate:"required"`
	// VM instance type
	// https://docs.microsoft.com/en-us/cli/azure/vm#list-sizes
	VmType string `json:"vm_type" yaml:"vm_type" validate:"required"`
	// SSHKeyPath specifies the location of the SSH private key to use for remote access
	SSHKeyPath string `json:"-" yaml:"key_path" validate:"required"`
	// AuthorizedKeysPath specifies ssh/authorized_keys file to be placed on remote machine
	AuthorizedKeysPath string `json:"ssh_authorized_keys_path" yaml:"authorized_keys_path" validate:"required"`
	// SSHUser defines SSH user used to connect to the provisioned machines
	SSHUser string `json:"ssh_user" yaml:"ssh_user" validate:"required"`
	// DockerDevice block device for docker data - set to /dev/sdd
	DockerDevice string `json:"docker_device" yaml:"docker_device" validate:"required"`
}
