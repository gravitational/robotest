# AMI IDs are region specific
# note each AMI is coming with preset username

variable "os" {
    default = "ubuntu"
}

variable "ami" {
  default = {
    us-east-1-ubuntu      = "ami-4a83175c"
    us-east-1-rhel7       = "ami-2051294a"
    us-east-1-centos7     = "ami-6d1c2007"
  }
}