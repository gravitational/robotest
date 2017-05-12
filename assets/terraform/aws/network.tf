# ALL UDP and TCP traffic is allowed within the security group
resource "aws_security_group" "cluster" {
    tags {
        Name = "${var.cluster_name}"
    }

    # Admin gravity site for testing
    ingress {
        from_port   = 32009
        to_port     = 32009
        protocol    = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    # SSH access from anywhere
    ingress {
        from_port   = 22
        to_port     = 22
        protocol    = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    # install wizard
    ingress {
        from_port   = 61009
        to_port     = 61009
        protocol    = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    ingress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        self = true
    }

    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }
}