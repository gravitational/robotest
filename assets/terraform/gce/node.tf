#
# Virtual Machine node
#

resource "google_compute_instance_group" "robotest" {
  description = "Instance group controlling instances of a single robotest cluster"
  name        = "${var.cluster_name}-grp"
  zone        = "${var.zone}"
  network     = "${google_compute_network.robotest.self_link}"
  instances   = ["${google_compute_instance.node.*.self_link}"]
}

resource "google_compute_instance" "node" {
  description  = "Instance is a single robotest cluster node"
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
    # Enable OS login using IAM roles
    enable_oslogin = "true"

    # sshKeys controls access to an instance using a custom SSH key
    sshKeys = "${var.ssh_user}:${file("${var.ssh_key}")}"
  }

  metadata_startup_script = "${data.template_file.bootstrap.rendered}"

  boot_disk {
    initialize_params {
      image = "${element(var.oss, var.os)}"
      size  = "64"
      type  = "${var.disk_type}"
    }

    auto_delete = "true"
  }

  attach_disk {
    source = "${google_compute_disk.etcd.*.self_link[count.index]}"
    mode   = "READ_WRITE"
  }

  attach_disk {
    source = "${google_compute_disk.docker.*.self_link[count.index]}"
    mode   = "READ_WRITE"
  }

  can_ip_forward = true
}

resource "google_compute_disk" "etcd" {
  count = "${var.nodes}"
  name  = "${var.cluster_name}-disk-etcd-${count.index}"
  type  = "pd-ssd"
  zone  = "${var.zone}"
  size  = "64"

  labels {
    cluster = "${var.cluster_name}"
  }
}

resource "google_compute_disk" "docker" {
  count = "${var.nodes}"
  name  = "${var.cluster_name}-disk-docker-${count.index}"
  type  = "pd-ssd"
  zone  = "${var.zone}"
  size  = "64"

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

