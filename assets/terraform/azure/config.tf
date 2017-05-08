# 
# Uses Azure Resource Manager provider
#   https://www.terraform.io/docs/providers/azurerm
# 
variable "azure_subscription_id"{ }
variable "azure_client_id"      { }
variable "azure_client_secret"  { }
variable "azure_tenant_id"      { }

variable "azure_resource_group" { default = "robotest-${uuid()}" }

variable "azure_location"       { }
variable "azure_vm_type"        { }

variable "ssh_key_path"         { }
variable "ssh_user"				{ default = "robotest" }

variable "nodes"				{ default = 2 }

# 
# Access credentials:
#   https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal
# 
provider "azurerm" {
  subscription_id = "${var.subscription_id}"
  client_id       = "${var.client_id}"
  client_secret   = "${var.client_secret}"
  tenant_id       = "${var.tenant_id}"
}

#
# everything is grouped under that resource group
# deleting it is a handy way to clean up 
# 
resource "azurerm_resource_group" "robotest" {
  name     = "${var.azure_resource_group}"
  location = "${var.azure_location}"
}
