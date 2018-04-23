#
# Output Variables
#

output "private_ips" {
  value = "${join(" ", google_compute_instance.node.*.network_interface.0.address)}"
}

output "public_ips" {
  value = "${join(" ", google_compute_instance.node.*.network_interface.0.access_config.0.assigned_nat_ip)}"
}
