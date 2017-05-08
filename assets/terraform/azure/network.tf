# 
# NETWORK
# 

resource "azurerm_network_security_group" "robotest" {
  name                = "robotest"
  location            = "${var.azure_location}"
  resource_group_name = "${azurerm_resource_group.robotest.name}"
  
  #
  # some things are case-sensitive. if you put Tcp instead of TCP, it won't work, but will render OK on GUI
  # 
  security_rule {
    name                       = "AdminGravitySite"
    priority                   = 1000
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "TCP"
    source_port_range          = "*"
    destination_port_range     = 3209
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "SSH"
    priority                   = 1010
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "TCP"
    source_port_range          = "*"
    destination_port_range     = 22
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "InstallWizard"
    priority                   = 1020
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "TCP"
    source_port_range          = "*"
    destination_port_range     = 61009
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

resource "azurerm_virtual_network" "robotest" {
  name                = "robotest"
  address_space       = ["10.40.0.0/16"]
  location            = "${var.azure_location}"
  resource_group_name = "${azurerm_resource_group.robotest.name}"
}

resource "azurerm_subnet" "robotest_a" {
  name                      = "robotest_10_40_2_0"
  resource_group_name       = "${azurerm_resource_group.robotest.name}"
  virtual_network_name      = "${azurerm_virtual_network.robotest.name}"
  address_prefix            = "10.40.2.0/24"
  # 
  # FIXME: azure API would hang indefinitely when security group is put here
  #        likely related to https://github.com/hashicorp/terraform/pull/9648
  #
  # network_security_group_id = "${azurerm_network_security_group.robotest.id}"
}

