
#
# Output Variables
# 

output "private_ips" {
  value = "${join(" ", azurerm_network_interface.node.*.private_ip_address)}"
}

output "public_ips" {
  value = "${join(" ", data.azurerm_public_ip.node.*.ip_address)}"
}
