#
# Output Variables
#

output "private_ips" {
  value = [google_compute_instance.node.*.network_interface.0.network_ip]
}

output "public_ips" {
  value = [google_compute_instance.node.*.network_interface.0.access_config.0.nat_ip]
}

# output "pod_cidr_blocks" {
#   value = ["${google_compute_instance.node.*.network_interface.0.alias_ip_range.0.ip_cidr_range}"]
# }
# 
# output "service_cidr_blocks" {
#   value = ["${google_compute_instance.node.*.network_interface.0.alias_ip_range.1.ip_cidr_range}"]
# }
