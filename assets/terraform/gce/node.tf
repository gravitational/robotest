#
# Virtual Machine node
#

resource "google_compute_instance_group" "robotest" {
  description = "Instance group controlling instances of a single robotest cluster"
  name        = "${var.node_tag}-node-group"
  zone        = "${local.zone}"
  network     = "${data.google_compute_network.robotest.self_link}"
  instances   = ["${google_compute_instance.node.*.self_link}"]
}

resource "google_compute_instance" "node" {
  description  = "Instance is a single robotest cluster node"
  count        = "${var.nodes}"
  name         = "${var.node_tag}-node-${count.index}"
  machine_type = "${var.vm_type}"
  zone         = "${local.zone}"

  tags = [
    "robotest",
    "${var.node_tag}-node-${count.index}",
  ]

  labels {
    cluster = "${var.node_tag}"
  }

  network_interface {
    subnetwork = "${data.google_compute_subnetwork.robotest.self_link}"

    access_config {
      # Ephemeral IP
    }

    # # https://www.terraform.io/docs/providers/google/r/compute_instance.html#alias_ip_range
    # # https://cloud.google.com/vpc/docs/alias-ip#key_benefits_of_alias_ip_ranges
    # #
    # # The main benefit of alias IP ranges is that routes are installed transparently and
    # # route quotas need not be taken into account.
    # # The only issue is marrying that to flannel (or doing away w/ it on GCE)
    # # --pod-network-cidr=
    # alias_ip_range {
    #   ip_cidr_range = "/26"
    # }

    # # --service-cidr=
    # alias_ip_range {
    #   ip_cidr_range = "/24"
    # }
  }

  metadata {
    # Enable OS login using IAM roles
    enable-oslogin = "true"

    # ssh-keys controls access to an instance using a custom SSH key
    # See: https://cloud.google.com/compute/docs/instances/adding-removing-ssh-keys#instance-only
    ssh-keys = "${var.os_user}:${file("${var.ssh_pub_key_path}")}"
  }

  metadata_startup_script = "${data.template_file.bootstrap.rendered}"

  min_cpu_platform = "Intel Skylake"

  boot_disk {
    initialize_params {
      image = "${lookup(var.oss, var.os)}"
      size  = 64
      type  = "${var.disk_type}"
    }

    auto_delete = true
  }

  attached_disk {
    source = "${google_compute_disk.etcd.*.self_link[count.index]}"
    mode   = "READ_WRITE"
  }

  attached_disk {
    source = "${google_compute_disk.docker.*.self_link[count.index]}"
    mode   = "READ_WRITE"
  }

  service_account {
    # TODO: consider using robotest-specific service account instead of
    # the default service account
    scopes = [
      "compute-rw",
      "service-control",
      "storage-ro",
    ]
  }

  scheduling {
    # https://cloud.google.com/compute/docs/instances/preemptible
    # This is a spot instance
    preemptible = true

    # If preempted, the test will be retried with a new configuration
    automatic_restart = false
  }
}

resource "google_compute_disk" "etcd" {
  count = "${var.nodes}"
  name  = "${var.node_tag}-disk-etcd-${count.index}"
  type  = "${var.disk_type}"
  zone  = "${local.zone}"
  size  = 50

  labels {
    cluster = "${var.node_tag}"
  }
}

resource "google_compute_disk" "docker" {
  # TODO: make docker disk optional
  # count = "${var.devicemapper_used ? var.nodes : 0}"
  count = "${var.nodes}"

  name = "${var.node_tag}-disk-docker-${count.index}"
  type = "${var.disk_type}"
  zone = "${local.zone}"
  size = 50

  labels {
    cluster = "${var.node_tag}"
  }
}

data "template_file" "bootstrap" {
  template = "${file("./bootstrap/${element(split(":",var.os),0)}.sh")}"

  vars {
    os_user     = "${var.os_user}"
    ssh_pub_key = "${file("${var.ssh_pub_key_path}")}"
  }
}
