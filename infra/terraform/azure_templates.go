package terraform

import "text/template"

const (
	tfVmFilename   = "vm.tf"
	tfDataFilename = "data.tf"
)

var tfVmImageTemplate = template.Must(template.New("vm_install").Parse(`
data "azurerm_image" "node" {
  count                 = "${var.nodes}"
  name                  = "node-${count.index}"
  resource_group_name   = "{{.ResourceGroup}}"
}

resource "azurerm_virtual_machine" "node" {
  count                 = "${var.nodes}"
  name                  = "node-${count.index}"
  location              = "${var.location}"
  resource_group_name   = "${azurerm_resource_group.robotest.name}"
  network_interface_ids = ["${azurerm_network_interface.node.*.id[count.index]}"]
  vm_size               = "${var.vm_type}"

  delete_os_disk_on_termination    = "true"
  delete_data_disks_on_termination = "true"

  storage_image_reference {
    id = "${data.azurerm_image.node.*.id[count.index]}"
  }

  storage_os_disk {
    name                = "node-os-${count.index}"
    caching             = "ReadWrite"
    create_option       = "FromImage"
    managed_disk_type   = "Premium_LRS"
    disk_size_gb        = "64"
  }

  os_profile {
    computer_name  = "node-${count.index}"
    # REQUIRED ...
    admin_username = "${var.ssh_user}"
    admin_password = "${var.random_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = true    
    ssh_keys = {
        path = "/home/${var.ssh_user}/.ssh/authorized_keys"
        key_data = "${file("${var.ssh_authorized_keys_path}")}"
    }
  }

  storage_data_disk {
    name              = "node-etcd-${count.index}"
    managed_disk_type = "Premium_LRS"
    create_option     = "FromImage"
    lun               = 0
    disk_size_gb      = "64"
  }

  storage_data_disk {
    name              = "node-docker-${count.index}"
    managed_disk_type = "Premium_LRS"
    create_option     = "FromImage"
    lun               = 1
    disk_size_gb      = "64"
  }

}
`))

var tfVmInstallTemplate = template.Must(template.New("vm_image").Parse(`
resource "azurerm_virtual_machine" "node" {
  count                 = "${var.nodes}"
  name                  = "node-${count.index}"
  location              = "${var.location}"
  resource_group_name   = "${azurerm_resource_group.robotest.name}"
  network_interface_ids = ["${azurerm_network_interface.node.*.id[count.index]}"]
  vm_size               = "${var.vm_type}"

  delete_os_disk_on_termination    = "true"
  delete_data_disks_on_termination = "true"

  storage_image_reference {
    publisher = "${lookup(var.os_publisher, element(split(":",var.os),0))}"
    offer     = "${lookup(var.os_offer,     element(split(":",var.os),0))}"
    sku       = "${lookup(var.os_sku,       var.os)}"
    version   = "${lookup(var.os_version,   var.os)}"
  }

  storage_os_disk {
    name                = "node-os-${count.index}"
    caching             = "ReadWrite"
    create_option       = "FromImage"
    managed_disk_type   = "Premium_LRS"
    disk_size_gb        = "64"
  }

  os_profile {
    computer_name  = "node-${count.index}"
    # REQUIRED ...
    admin_username = "${var.ssh_user}"
    admin_password = "${var.random_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = true    
    ssh_keys = {
        path = "/home/${var.ssh_user}/.ssh/authorized_keys"
        key_data = "${file("${var.ssh_authorized_keys_path}")}"
    }
  }

  storage_data_disk {
    name              = "node-etcd-${count.index}"
    managed_disk_type = "Premium_LRS"
    create_option     = "Empty"
    lun               = 0
    disk_size_gb      = "64"
  }

  storage_data_disk {
    name              = "node-docker-${count.index}"
    managed_disk_type = "Premium_LRS"
    create_option     = "Empty"
    lun               = 1
    disk_size_gb      = "64"
  }
}
`))
