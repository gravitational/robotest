#
# OS configuration
#

variable "os_images" {
  description = "Map of supported Linux distributions"
  type        = "map"

  default = {
    # os -> {project}/{image}
    "ubuntu:16"     = "ubuntu-16.04.img"
    "ubuntu:18"     = "ubuntu-18.04.img"
    "ubuntu:latest" = "ubuntu-latest.img"
    "redhat:7"      = "redhat-7.qcow2"
    "centos:7"      = "centos-7.qcow2"
    "centos:latest" = "centos-latest.qcow2"
    "debian:8"      = "debian-8.qcow2"
    "debian:9"      = "debian-9.qcow2"
    "debian:latest" = "debian-latest.qcow2"
    "suse:12"       = "suse-12.qcow2"
    "suse:15"       = "suse-15.qcow2"
    "suse:latest"   = "suse-latest.qcow2"
  }
}

