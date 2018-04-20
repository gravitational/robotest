#
# Virtual Machine node
#

resource "google_compute_instance_group" "robotest" {
  name    = "${var.cluster_name}-grp"
  zone    = "${var.zone}"
  network = "${google_compute_network.robotest.self_link}"
}

resource "google_compute_instance_template" "node" {
  count        = "${var.nodes}"
  name         = "${var.cluster_name}-node-${count.index}"
  machine_type = "${var.vm_type}"
  zone         = "${var.zone}"

  tags = [
    "robotest",
    "${var.cluster_name}-node-${count.index}",
  ]

  labels {
    cluster = "${var.cluster_name}"
  }

  network_interface {
    network = "${data.google_compute_subnetwork.robotest.self_link}"

    access_config {
      # Ephemeral IP
    }
  }

  metadata {
    sshKeys = "${var.ssh_user}:${file("${var.ssh_key_data}")}"
  }

  metadata_startup_script = "${data.template_file.bootstrap.rendered}"

  disk {
    disk_name    = "${var.cluster_name}-os-${count.index}"
    source_image = "${element(var.oss, var.os)}"
    disk_type    = "${var.disk_type}"
    disk_size_gb = "64"
    mode         = "READ_WRITE"
    auto_delete  = "true"
    boot         = "true"
  }

  disk {
    source      = "${google_compute_disk.etcd.self_link}"
    disk_name   = "${var.cluster_name}-node-etcd-${count.index}"
    mode        = "READ_WRITE"
    auto_delete = "true"
  }

  disk {
    source      = "${google_compute_disk.docker.self_link}"
    disk_name   = "node-docker-${count.index}"
    mode        = "READ_WRITE"
    auto_delete = "true"
  }

  can_ip_forward = true
}

resource "google_compute_disk" "etcd" {
  name = "${var.cluster_name}-disk-etcd"
  type = "pd-ssd"
  zone = "${var.zone}"
  size = "64"

  labels {
    cluster = "${var.cluster_name}"
  }
}

resource "google_compute_disk" "docker" {
  name = "${var.cluster_name}-disk-docker"
  type = "pd-ssd"
  zone = "${var.zone}"
  size = "64"

  labels {
    cluster = "${var.cluster_name}"
  }
}

data "template_file" "bootstrap" {
  template = "${file("./bootstrap/${element(split(":",var.os),0)}.sh")}"

  vars {
    service_uid = "${vars.service_uid}"
    service_gid = "${vars.service_gid}"
  }
}

# # FIXME: is this the way to properly read the address attribute
# # of a compute instance?
# data "google_compute_address" "node" {
#   name = "node"
# 
#   # count = "${var.nodes}"
# }

