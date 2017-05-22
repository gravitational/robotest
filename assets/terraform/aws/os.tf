# AMI IDs are region specific
# note each AMI is coming with preset username

variable "ami" {
  default = {
    us-east-1-ubuntu = "ami-4a83175c"
    us-east-1-redhat = "ami-2051294a"
    us-east-1-centos = "ami-6d1c2007"
    us-east-1-debian = "ami-b14ba7a7"
  }
}