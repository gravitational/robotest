# ALL UDP and TCP traffic is allowed within the security group
resource "aws_security_group" "cluster" {
    tags {
        Name = "${var.cluster_name}"
    }

    # SSH
    ingress {
        from_port = 22
        to_port = 22
        protocol = "tcp"
        self = true
        cidr_blocks = ["0.0.0.0/0"]
    }

    # installer ports
    ingress {
        from_port = 61008
        to_port = 61010
        protocol = "tcp"
        self = true
        cidr_blocks = ["0.0.0.0/0"]
    }

    # installer ports
    ingress {
        from_port = 61022
        to_port = 61024
        protocol = "tcp"
        self = true
        cidr_blocks = ["0.0.0.0/0"]
    }

    # bandwidth checker (for pre-checks)
    ingress {
        from_port   = 4242
        to_port     = 4242
        protocol    = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    # k8s health check
    ingress {
        from_port = 10250
        to_port = 10250
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    # serf peer-to-peer
    ingress {
        from_port = 7496
        to_port = 7496
        protocol = "tcp"
        self = true
    }

    # kubernetes apiserver insecure
    ingress {
        from_port = 8080
        to_port = 8080
        protocol = "tcp"
        self = true
    }

    # kubernetes apiserver secure
    ingress {
        from_port = 6443
        to_port = 6443
        protocol = "tcp"
        self = true
    }

    # etcd peer-to-peer
    ingress {
        from_port = 2380
        to_port = 2380
        protocol = "tcp"
        self = true
    }

    # etcd client
    ingress {
        from_port = 2379
        to_port = 2379
        protocol = "tcp"
        self = true
    }

    # etcd legacy client
    ingress {
        from_port = 4001
        to_port = 4001
        protocol = "tcp"
        self = true
    }

    # etcd legacy peer-to-peer
    ingress {
        from_port = 7001
        to_port = 7001
        protocol = "tcp"
        self = true
    }

    # docker registry
    ingress {
        from_port = 5000
        to_port = 5000
        protocol = "tcp"
        self = true
    }

    # overlay networking
    ingress {
        from_port = 8472
        to_port = 8472
        protocol = "udp"
        self = true
    }

    # gravity profiling port
    ingress {
        from_port = 6060
        to_port = 6060
        protocol = "tcp"
        self = true
    }

    # teleport services (SSH server, proxy service, proxy tunnel, auth service)
    ingress {
        from_port = 3022
        to_port = 3025
        protocol = "tcp"
        self = true
    }

    # teleport web
    ingress {
        from_port = 3080
        to_port = 3080
        protocol = "tcp"
        self = true
    }

    # internal k8s services
    ingress {
        from_port = 30000
        to_port = 32767
        protocol = "tcp"
        self = true
        cidr_blocks = ["0.0.0.0/0"]
    }

    # kubernetes kubelet
    ingress {
        from_port = 10248
        to_port = 10249
        protocol = "tcp"
        self = true
    }

    # kubernetes kubelet
    ingress {
        from_port = 10255
        to_port = 10255
        protocol = "tcp"
        self = true
    }

    # gravity services (ops service, pack service, health test)
    ingress {
        from_port = 3008
        to_port = 3010
        protocol = "tcp"
        self = true
    }

    # planet agent RPC
    ingress {
        from_port = 7575
        to_port = 7575
        protocol = "tcp"
        self = true
    }

    # serf RPC
    ingress {
        from_port = 7373
        to_port = 7373
        protocol = "tcp"
        self = true
    }

    egress {
      from_port = 0
      to_port = 0
      protocol = "-1"
      cidr_blocks = ["0.0.0.0/0"]
    }    
}