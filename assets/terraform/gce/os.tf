#
# OS configuration
#
# https://cloud.google.com/compute/docs/images
#

variable "oss" {
  description = "Map of supported Linux distributions"
  type        = "map"

  default = {
    # os -> {project}/{image}
    "ubuntu:16" = "ubuntu-os-cloud/ubuntu-1604-xenial-v20180405"
    "ubuntu:17" = "ubuntu-os-cloud/ubuntu-1710-artful-v20180405"
    "redhat:7"  = "rhel-cloud/rhel-7-v20180401"
    "centos:7"  = "centos-cloud/centos-7-v20180401"
    "debian:8"  = "debian-cloud/debian-8-jessie-v20180401"
    "debian:9"  = "debian-cloud/debian-9-stretch-v20180401"
    "suse:12"   = "suse-cloud/sles-12-sp3-v20180214"
  }
}
