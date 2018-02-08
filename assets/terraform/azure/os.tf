#
# OS configuration
#
# https://docs.microsoft.com/en-us/azure/virtual-machines/linux/cli-ps-findimage
#

variable "os_publisher" {
    type = "map"

    default = {
        "ubuntu" = "Canonical"
        "redhat" = "RedHat"
        "centos" = "OpenLogic"
        "debian" = "credativ"
        "suse"   = "SUSE"
    }
}

variable "os_offer" {
    type = "map"

    default = {
        "ubuntu" = "UbuntuServer"
        "redhat" = "RHEL"
        "centos" = "CentOS"
        "debian" = "Debian"
        "suse"   = "SLES"
    }
}

variable "os_sku" {
    type = "map"

    default = {
        "ubuntu:latest" = "16.04-LTS"
        "redhat:7.2" = "7.2"
        "redhat:7.3" = "7.3"
        "redhat:7.4" = "7-RAW-CI"
        "centos:7.2" = "7.2"
        "centos:7.3" = "7.3"
        "centos:7.4" = "7-CI"
        "debian"     = "8"
        "suse"       = "12-SP3"
    }
}

variable "os_version" {
    type = "map"

    default = {
        "ubuntu:latest"     = "16.04.201708151"
        "redhat:7.3" = "latest"
        "redhat:7.2" = "latest"
        "redhat:7.4" = "latest"
        "centos:7.4" = "latest"
        "centos:7.3" = "7.3.20170925"
        "centos:7.2" = "7.2.20170517"
        "debian"     = "latest"
        "suse"       = "2017.09.07"
    }
}
