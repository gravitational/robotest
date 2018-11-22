# 
# Uses Azure Resource Manager provider
#   https://www.terraform.io/docs/providers/azurerm
# 
variable "subscription_id" {}

variable "client_id" {}
variable "client_secret" {}
variable "tenant_id" {}

variable "azure_resource_group" {}

variable "location" {}
variable "vm_type" {}

variable "ssh_authorized_keys_path" {}

variable "ssh_user" {
  default = "robotest"
}

variable "nodes" {}

variable "os" {
  description = "ubuntu | redhat | centos | debian"
}

variable random_password {}

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
  location = "${var.location}"
}
