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
		"centos" = "7.3"
		"debian" = "8"
	}
}

variable "os_version" {
	type = "map"

	default = {
		"ubuntu" = "latest"
		"redhat" = "7.3.2017090723"
		"centos" = "latest"
		"debian" = "latest"
	}
}
