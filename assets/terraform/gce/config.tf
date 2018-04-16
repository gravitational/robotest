# 
# Google provider
#   https://www.terraform.io/docs/providers/google/index.html
# 

variable "credentials" {
  description = "JSON-encoded access credentials"
}
variable "project" {
  description = "Project name"
  default = "kubeadm"
}
variable "region" {
  description = "Cloud region"
  default = "us-central1"
}
variable "zone" {
  description = "Cloud zone"
  default = "us-central1-a"
}

variable "resource_group" {
  description = "Name of the instance group"
}
variable "vm_type" {
  description = "Type of VM to provision"
  default = "f1-micro"
}
variable "location" { }
variable "tags" {
  description = "List of tags to assign to an instance"
  type = "map"
}

variable "ssh_authorized_keys_path"  { }
variable "ssh_user"				{ default = "robotest" }

# Number of nodes to provision
variable "nodes" { }

variable "os" { 
	# description = "ubuntu | redhat | centos | debian | sles"
	description = "ubuntu | redhat | centos | debian"
}

variable random_password { }

# 
# Access credentials:
#   https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal
# 
provider "google" {
  credentials = "${file("${var.credentials}")}"
  project = "${var.project}"
  region = "${var.region}"
  zone = "${var.zone}"
}

#
# everything is grouped under an instance group.
# deleting the group results in all resources being deleted.
# 
resource "google_compute_instance_group" "robotest" {
  name     = "${var.resource_group}"
  location = "${var.location}"
}
