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
	}
}

variable "os_offer" {
	type = "map"

	default = {
		"ubuntu" = "UbuntuServer"
		"redhat" = "RHEL"
		"centos" = "CentOS"
		"debian" = "Debian"
	}
}

variable "os_sku" {
	type = "map"

	default = {
		"ubuntu" = "16.04-LTS"
		"redhat" = "7.3"
		"redhat:7.2" = "7.2"
		"redhat:7.3" = "7.3"
		"redhat:7.4" = "7-RAW"
		"centos" = "7.3"
		"centos:7.2" = "7.2"
		"centos:7.3" = "7.3"
		"debian" = "8"
	}
}

variable "os_version" {
	type = "map"

	default = {
		"ubuntu" 	 = "16.04.201708151"
		"redhat" 	 = "7.3.2017090723"
		"redhat:7.3" = "7.3.2017090723"
		"redhat:7.2" = "latest"
		"redhat:7.4" = "7.4.2017080923"
		"centos" 	 = "latest"
		"centos:7.4" = "7.4.20170919"
		"centos:7.3" = "7.3.20170925"
		"centos:7.2" = "7.2.20170517"
		"debian" 	 = "latest"
	}
}
