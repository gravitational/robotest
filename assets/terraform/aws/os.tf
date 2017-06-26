# AMI IDs are region specific
# note each AMI is coming with preset username

variable "ami" {
  default = {
    ubuntu = "ami-4a83175c"
    redhat = "ami-2051294a"
    centos = "ami-6d1c2007"
    debian = "ami-b14ba7a7"
  }
}