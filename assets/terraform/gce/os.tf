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
    "ubuntu:16"     = "ubuntu-os-cloud/ubuntu-1604-xenial-v20190816"
    "ubuntu:18"     = "ubuntu-os-cloud/ubuntu-1804-bionic-v20190813a"
    "ubuntu:latest" = "ubuntu-os-cloud/ubuntu-1804-bionic-v20190813a"
    "redhat:7"      = "rhel-cloud/rhel-7-v20190813"
    "centos:7"      = "centos-cloud/centos-7-v20190813"
    "debian:8"      = "debian-cloud/debian-8-jessie-v20180401"
    "debian:9"      = "debian-cloud/debian-9-stretch-v20190813"
    "debian:latest" = "debian-cloud/debian-9-stretch-v20190813"
    "suse:12"       = "suse-cloud/sles-12-sp4-v20190617"
    "suse:15"       = "suse-cloud/sles-15-sp1-v20190828"
    "suse:latest"   = "suse-cloud/sles-15-sp1-v20190828"
  }
}

