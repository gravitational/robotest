#
# OS configuration
#
variable "os" {
	type = "map"

	default = {
		"publisher" = "Canonical"
		"offer"     = "UbuntuServer"
		"sku"       = "16.04-LTS"
		"version"   = "latest"
	}
}
