resource "aws_instance" "node" {
    ami = "${lookup(var.ami, var.os)}"
    instance_type = "${var.instance_type}"
    associate_public_ip_address = true
    source_dest_check = "false"
    ebs_optimized = true
    security_groups = ["${aws_security_group.cluster.name}"]
    key_name = "${var.key_pair}"
    placement_group = "${aws_placement_group.cluster.id}"
    count = "${var.nodes}"

    tags {
        Name = "${var.cluster_name}"
        Origin = "robotest"
    }

    user_data = "${file("./bootstrap/${var.os}.sh")}"

    # OS
    # /var/lib/gravity device
    # /var/lib/data device
    root_block_device {
        volume_type = "gp2"
        volume_size = "60"
        delete_on_termination = true
    }

    # gravity/docker data device
    ebs_block_device = {
        volume_type = "gp2"
        volume_size = "80"
        device_name = "${var.docker_device}"
        delete_on_termination = true
    }

    # etcd device
    ebs_block_device = {
        volume_type = "io1"
        iops = 1500
        volume_size = "30"
        device_name = "/dev/xvdc"
        delete_on_termination = true
    }
}