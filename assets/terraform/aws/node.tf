resource "aws_instance" "node" {
    ami = "${lookup(var.ami, "${var.region}-${var.os}")}"
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

    user_data = "${file("./bootstrap.sh")}"

    root_block_device {
        delete_on_termination = true
        volume_type = "io1"
        volume_size = "50"
        iops = 500
    }

    # etc and other data device
    ebs_block_device = {
        volume_size = "500"
        volume_type = "io1"
        device_name = "/dev/xvde"
        iops = 3000
        delete_on_termination = true
    }

    # gravity/docker data device
    ebs_block_device = {
        volume_size = "500"
        volume_type = "io1"
        device_name = "/dev/xvdd"
        iops = 3000
        delete_on_termination = true
    }    
}