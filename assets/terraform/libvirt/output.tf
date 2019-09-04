#
# Output Variables
#

output "private_ips" {
  value = "${libvirt_domain.domain-gravity.*.network_interface.0.addresses.0}"
}

output "public_ips" {
  value = "${libvirt_domain.domain-gravity.*.network_interface.0.addresses.0}"
}