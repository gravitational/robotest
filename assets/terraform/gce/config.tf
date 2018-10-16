#
# Google provider
#   https://www.terraform.io/docs/providers/google/index.html
#

variable "credentials" {
  description = "JSON-encoded access credentials"
}

variable "project" {
  description = "Project to deploy to, if not set the default provider project is used."
  default     = "kubeadm-167321"
}

variable "region" {
  description = "Region for cluster resources"
  default     = "us-central1"
}

variable "zone" {
  description = "Zone for cluster resources."
  default     = "us-central1-a"
}

variable "cluster_name" {
  description = "Name of the robotest cluster"
}

variable "node_tag" {
  description = "GCE-friendly cluster name to use as a prefix for resources."
}

variable "instance_type" {
  description = "Type of VM to provision. See https://cloud.google.com/compute/docs/machine-types"
  default     = "n1-standard-1"
}

variable "os_user" {
  description = "SSH user to login onto nodes"
  default     = "robotest"
}

variable "ssh_key_path" {
  description = "Path to the public SSH key."
}

variable "nodes" {
  description = "Number of nodes to provision"
}

variable "os" {
  description = "Linux distribution as name:version, i.e. debian:9"
}

variable "disk_type" {
  description = "Disk type for VM. See https://cloud.google.com/compute/docs/disks"
  default     = "pd-ssd"
}

provider "google" {
  credentials = "${file("${var.credentials}")}"
  project     = "${var.project}"
  region      = "${var.region}"
}

# List zones available in a region
data "google_compute_zones" "available" {
  region = "${var.region}"
}

resource "random_shuffle" "zones" {
  input        = ["${data.google_compute_zones.available.names}"]
  result_count = 1
}

locals {
  zone = "${random_shuffle.zones.result[0]}"
}
