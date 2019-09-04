#
# libvirt provider
#   
#

variable "ssh_user" {
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

variable "image_name" {
  description = "OS image file"
  type        = "string"
  default     = "ubuntu-16.04-server-cloudimg-amd64-disk1.img"
}

variable "gravity_dir_size" {
  description = "Disk size for gravity directory"
  type        = "string"
  default     = "12000000000"
}

variable "memory_size" {
  description = "Node memory size"
  type    = "string"
  default = "2048"
}

variable "cpu_count" {
  description = "Virtual CPU count"
  type        = "string"
  default     = "1"
}

provider "libvirt" {
  uri = "qemu:///system"
}