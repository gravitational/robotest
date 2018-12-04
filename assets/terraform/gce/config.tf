#
# Google provider
#   https://www.terraform.io/docs/providers/google/index.html
#

variable "credentials" {
  description = "JSON-encoded access credentials"
  type        = "string"
}

variable "project" {
  description = "Project to deploy to, if not set the default provider project is used."
  type        = "string"
  default     = "kubeadm-167321"
}

variable "region" {
  description = "Region for cluster resources"
  type        = "string"
  default     = "us-central1"
}

variable "zone" {
  description = "Zone for cluster resources."
  type        = "string"
  default     = "us-central1-a"
}

variable "node_tag" {
  description = "GCE-friendly cluster name to use as a prefix for resources."
  type        = "string"
}

variable "vm_type" {
  description = "Type of VM to provision. See https://cloud.google.com/compute/docs/machine-types"
  type        = "string"
  default     = "n1-standard-1"
}

variable "os_user" {
  description = "SSH user to login onto nodes"
  type        = "string"
}

variable "ssh_pub_key_path" {
  description = "Path to the public SSH key."
  type        = "string"
}

variable "nodes" {
  description = "Number of nodes to provision"
  type        = "string"
  default     = 1
}

variable "os" {
  description = "Linux distribution as name:version, i.e. debian:9"
  type        = "string"
}

variable "disk_type" {
  description = "Disk type for VM. See https://cloud.google.com/compute/docs/disks"
  type        = "string"
  default     = "pd-ssd"
}

variable "preemptible" {
  description = "Whether to use preemptible VMs. See https://cloud.google.com/preemptible-vms"
  type        = "string"
  default     = "true"
}

variable "robotest_node_ip" {
  description = "Public IP address of the robotest controller node to add to sshguard's whitelist"
  type        = "string"
}

provider "google" {
  credentials = "${file("${var.credentials}")}"
  project     = "${var.project}"
  region      = "${var.region}"
  version     = "~> 1.19"
}

provider "random" {
  version = "~> 2.0"
}

provider "template" {
  version = "~> 1.0"
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
