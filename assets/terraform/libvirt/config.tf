#
# libvirt provider
#   

variable "ssh_user" {
  description = "SSH user to login onto nodes"
  type        = "string"
}

variable "ssh_pub_key_path" {
  description = "Path to the public SSH key"
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

variable "disk_size" {
  description = "Main disk size in bytes"
  type        = "string"
  default     = "40000000000" # ~40GB
}

variable "memory" {
  description = "Node memory size in MB"
  type        = "string"
  default     = "2048" # 2GB
}

variable "cpu" {
  description = "Virtual CPU count"
  type        = "string"
  default     = "2"
}

provider "libvirt" {
  uri = "qemu:///system"
}