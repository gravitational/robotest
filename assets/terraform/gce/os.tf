#
# OS configuration
#
# https://cloud.google.com/compute/docs/images#os-compute-support
#

variable "oss" {
  description = "Map of supported Linux distributions"
  type        = map(string)

  default = {
    # os -> {project}/{image}
    "ubuntu:16"     = "ubuntu-os-cloud/ubuntu-1604-xenial-v20200407"
    "ubuntu:18"     = "ubuntu-os-cloud/ubuntu-1804-bionic-v20200414"
    "ubuntu:19"     = "ubuntu-os-cloud/ubuntu-1904-eoan-v20200413a"
    "ubuntu:latest" = "ubuntu-os-cloud/ubuntu-1904-eoan-v20200413a"
    "redhat:7"      = "rhel-cloud/rhel-7-v20200420"
    "redhat:8"      = "rhel-cloud/rhel-8-v20200413"
    "centos:7"      = "centos-cloud/centos-7-v20200420"
    "centos:8"      = "centos-cloud/centos-8-v20200413"
    "debian:9"      = "debian-cloud/debian-9-stretch-v20200420"
    "debian:10"     = "debian-cloud/debian-10-buster-v20200413"
    "debian:latest" = "debian-cloud/debian-10-buster-v20200413"
    "suse:12"       = "suse-cloud/sles-12-sp5-v20200227"
    "suse:15"       = "suse-cloud/sles-15-sp1-v20200415"
    "suse:latest"   = "suse-cloud/sles-15-sp1-v20200415"
  }
}

