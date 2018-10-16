#
# Network
#

resource "google_compute_firewall" "ssh" {
  name    = "${var.node_tag}-allow-ssh"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports    = ["22", "61822"]
  }
}

resource "google_compute_firewall" "web" {
  name    = "${var.node_tag}-web-admin"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["32009"]
  }
}

resource "google_compute_firewall" "telekube_installer" {
  name    = "${var.node_tag}-installer"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["61008-61010", "61022-61024", "3008-3010", "6060"]
  }
}

resource "google_compute_firewall" "bandwidth" {
  name    = "${var.node_tag}-bandwidth-test"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["4242"]
  }
}

resource "google_compute_firewall" "kubelet" {
  name    = "${var.node_tag}-kubelet"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["10249", "10250", "10255"]
  }
}

resource "google_compute_firewall" "serf" {
  name    = "${var.node_tag}-serf-peer-network"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["7496", "7373"]
  }
}

resource "google_compute_firewall" "k8s_api" {
  name    = "${var.node_tag}-kubernetes-api-server"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["8080", "6443"]
  }
}

resource "google_compute_firewall" "k8s" {
  name    = "${var.node_tag}-kubernetes-internal-services"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["30000-32767"]
  }
}

resource "google_compute_firewall" "etcd" {
  name    = "${var.node_tag}-etcd"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["2379", "2380", "4001", "7001"]
  }
}

resource "google_compute_firewall" "registry" {
  name    = "${var.node_tag}-docker-registry"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["5000"]
  }
}

resource "google_compute_firewall" "overlay_network" {
  name    = "${var.node_tag}-overlay-network"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["8472"]
  }
}

resource "google_compute_firewall" "teleport" {
  name    = "${var.node_tag}-teleport-services"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["3022-3025"]
  }
}

resource "google_compute_firewall" "planet" {
  name    = "${var.node_tag}-planet-rpc"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["7575"]
  }
}

resource "google_compute_firewall" "ntp" {
  name    = "${var.node_tag}-ntp"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "udp"
    ports    = ["123"]
  }
}

data "google_compute_network" "robotest" {
  name = "default"
}

# FIXME: propagate to gravity as `--pod-cidr` and `--service-cidr`
resource "google_compute_subnetwork" "pods" {
  network = "${google_compute_network.robotest.self_link}"

  secondary_ip_range {
    range_name    = "${var.node_tag}-pod-cidr"
    ip_cidr_range = "/26"                      # enough for 64 Pods
  }
}

resource "google_compute_subnetwork" "services" {
  network = "${google_compute_network.robotest.self_link}"

  secondary_ip_range {
    range_name    = "${var.node_tag}-service-cidr"
    ip_cidr_range = "/16"
  }
}
