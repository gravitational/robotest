#
# Google provider
#   https://www.terraform.io/docs/providers/google/index.html
#

variable "credentials" {
  description = "JSON-encoded access credentials"
}

variable "project" {
  description = "Project name"
  default     = "kubeadm"
}

variable "region" {
  description = "Cloud region"
  default     = "us-central1"
}

# variable "zone" {
#   description = "Cloud zone"
#   default     = "us-central1-a"
# }

variable "cluster_name" {
  description = "Name of the robotest cluster"
}

variable "vm_type" {
  description = "Type of VM to provision. See https://cloud.google.com/compute/docs/machine-types"
  default     = "n1-standard-1"
}

variable "tags" {
  description = "List of tags to assign to an instance"
  type        = "map"
}

variable "ssh_key_path" {
  description = "Path to the SSH key"
}

variable "ssh_user" {
  description = "SSH user to login onto nodes"
  default     = "robotest"
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

variable "service_uid" {
  description = "Service user ID to own state directory/file permissions"
  default     = "1000"
}

variable "service_gid" {
  description = "Service group ID to own state directory/file permissions"
  default     = "1000"
}

variable random_password {}

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
  input        = ["${data.google_compute_zones.available}"]
  result_count = 1
}

locals {
  zone = "${random_shuffle.zones.result[0]}"
}
