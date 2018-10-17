#
# OS configuration
#
# https://cloud.google.com/compute/docs/images#os-compute-support
#

variable "oss" {
  description = "Map of supported Linux distributions"
  type        = "map"

  default = {
    # os -> {project}/{image}
    "ubuntu:16"     = "ubuntu-os-cloud/ubuntu-1604-xenial-v20181004"
    "ubuntu:18"     = "ubuntu-os-cloud/ubuntu-1804-bionic-v20181003"
    "ubuntu:latest" = "ubuntu-os-cloud/ubuntu-1804-bionic-v20181003"
    "redhat:7"      = "rhel-cloud/rhel-7-v20181011"
    "centos:7"      = "centos-cloud/centos-7-v20181011"
    "debian:8"      = "debian-cloud/debian-8-jessie-v20180401"
    "debian:9"      = "debian-cloud/debian-9-stretch-v20181011"
    "debian:latest" = "debian-cloud/debian-9-stretch-v20181011"
    "suse:12"       = "suse-cloud/sles-12-sp3-v20180814"
    "suse:15"       = "suse-cloud/sles-15-v20180816"
    "suse:latest"   = "suse-cloud/sles-15-v20180816"
  }
}
