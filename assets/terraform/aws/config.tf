variable "access_key" {}
variable "secret_key" {}
variable "ssh_user" {}
variable "region" {}

variable "key_pair" {}

variable "cluster_name" { }

variable "nodes" {
	description = "number of nodes in cluster"
}

variable "os" { 
	description = "ubuntu | redhat | centos | debian"
}

variable "instance_type" {
  default = "c3.xlarge"
}

provider "aws" {
  access_key = "${var.access_key}"
  secret_key = "${var.secret_key}"
  region = "${var.region}"
}

resource "aws_placement_group" "cluster" {
  name = "${var.cluster_name}"
  strategy = "cluster"
}
