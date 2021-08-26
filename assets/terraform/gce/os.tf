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
    "ubuntu:16"     = "ubuntu-os-cloud/ubuntu-1604-xenial-v20200807"
    "ubuntu:18"     = "ubuntu-os-cloud/ubuntu-1804-bionic-v20200807"
    "ubuntu:20"     = "ubuntu-os-cloud/ubuntu-2004-focal-v20200729"
    "ubuntu:latest" = "ubuntu-os-cloud/ubuntu-2004-focal-v20200729"

    "redhat:7.8"    = "rhel-cloud/rhel-7-v20200910"
    "redhat:7.9"    = "rhel-cloud/rhel-7-v20210817"
    "redhat:7"      = "rhel-cloud/rhel-7-v20210817"
    "redhat:8.2"    = "rhel-cloud/rhel-8-v20200910"
    "redhat:8.3"    = "rhel-cloud/rhel-8-v20201112"
    "redhat:8.4"    = "rhel-cloud/rhel-8-v20210817"
    "redhat:8"      = "rhel-cloud/rhel-8-v20210817"

    "centos:7.8"    = "centos-cloud/centos-7-v20200910"
    "centos:7.9"    = "centos-cloud/centos-7-v20210817"
    "centos:7"      = "centos-cloud/centos-7-v20210817"
    "centos:8.2"    = "centos-cloud/centos-8-v20200910"
    "centos:8.3"    = "centos-cloud/centos-8-v20201112"
    "centos:8.4"    = "centos-cloud/centos-8-v20210817"
    "centos:8"      = "centos-cloud/centos-8-v20210817"

    "debian:8"      = "debian-cloud/debian-8-jessie-v20180611"
    "debian:9"      = "debian-cloud/debian-9-stretch-v20210817"
    "debian:10"     = "debian-cloud/debian-10-buster-v20210817"
    "debian:latest" = "debian-cloud/debian-10-buster-v20210817"

    # suse is an alias of sles for backwards compatibility, may be removed in 3.0
    "suse:12"       = "suse-cloud/sles-12-sp5-v20200916"
    "suse:15"       = "suse-cloud/sles-15-sp2-v20201014"
    "suse:latest"   = "suse-cloud/sles-15-sp2-v20201014"

    "sles:12-sp5"   = "suse-cloud/sles-12-sp5-v20200916"
    "sles:12"       = "suse-cloud/sles-12-sp5-v20200916"
    "sles:15-sp2"   = "suse-cloud/sles-15-sp2-v20201014"
    "sles:15"       = "suse-cloud/sles-15-sp2-v20201014"
  }
}

