#
# Virtual Machine node
#

resource "azurerm_public_ip" "node" {
  count                        = "${var.node_count}"
  name                         = "node-${count.index}"
  location                     = "${var.azure_location}"
  resource_group_name          = "${azurerm_resource_group.robotest.name}"
  public_ip_address_allocation = "dynamic"
}

resource "azurerm_network_interface" "node" {
  count                = "${var.node_count}"
  name                 = "node-${count.index}"
  location             = "${var.azure_location}"
  resource_group_name  = "${azurerm_resource_group.robotest.name}"
  enable_ip_forwarding = "true"
  network_security_group_id = "${azurerm_network_security_group.robotest.id}"

  ip_configuration {
    name                          = "ipconfig-${count.index}"
    subnet_id                     = "${azurerm_subnet.robotest_a.id}"
    private_ip_address_allocation = "dynamic"
    public_ip_address_id          = "${azurerm_public_ip.node.*.id[count.index]}"
  }
}

resource "azurerm_virtual_machine" "node" {
  count                 = "${var.node_count}"
  name                  = "node-${count.index}"
  location              = "${var.azure_location}"
  resource_group_name   = "${azurerm_resource_group.robotest.name}"
  network_interface_ids = ["${azurerm_network_interface.node.*.id[count.index]}"]
  vm_size               = "${var.azure_vm_type}"

  storage_image_reference {
    publisher = "${var.os["publisher"]}"
    offer     = "${var.os["offer"]}"
    sku       = "${var.os["sku"]}"
    version   = "${var.os["version"]}"
  }

  storage_os_disk {
    name                = "node-os-${count.index}"
    caching             = "ReadWrite"
    create_option       = "FromImage"
    managed_disk_type   = "Premium_LRS"
  }

  os_profile {
    computer_name  = "node-${count.index}"
    # REQUIRED ...
    admin_username = "${var.ssh_user}"
    admin_password = "Password1234!"
  }

  os_profile_linux_config {
    disable_password_authentication = true    
    ssh_keys = {
        path = "/home/${var.ssh_user}/.ssh/authorized_keys"
        key_data = "${file("${var.ssh_key_path}")}"
    }
  }

}
