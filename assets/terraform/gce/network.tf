#
# Network
#

resource "google_compute_firewall" "ssh" {
  name        = "${var.node_tag}-ssh"
  description = "SSH access"
  network     = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports    = ["22", "61822"]
  }
}

resource "google_compute_firewall" "web" {
  name        = "${var.node_tag}-web"
  description = "Web UI"
  network     = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["32009"]
  }
}

resource "google_compute_firewall" "installer" {
  name        = "${var.node_tag}-installer"
  description = "Internal installer ports"
  network     = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["61008-61010", "61022-61024", "3008-3010", "6060"]
  }
}

resource "google_compute_firewall" "bandwidth" {
  name        = "${var.node_tag}-bandwidth"
  description = "Internal network bandwidth preflight test port"
  network     = "${data.google_compute_network.robotest.self_link}"

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
    ports    = ["10248", "10250"]
  }
}

resource "google_compute_firewall" "serf" {
  name        = "${var.node_tag}-serf"
  description = "Internal serf gossip cluster"
  network     = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["7496", "7373"]
  }
}

resource "google_compute_firewall" "k8s" {
  name        = "${var.node_tag}-k8s"
  description = "Kubernetes API server and internal services"
  network     = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["6443", "30000-32767"]
  }
}

resource "google_compute_firewall" "etcd" {
  name        = "${var.node_tag}-etcd"
  description = "Internal etcd cluster"
  network     = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["2379", "2380", "4001", "7001"]
  }
}

resource "google_compute_firewall" "registry" {
  name        = "${var.node_tag}-registry"
  description = "Internal docker registry"
  network     = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["5000"]
  }
}

resource "google_compute_firewall" "overlay_network" {
  name        = "${var.node_tag}-overlay"
  description = "Flannel overlay network"
  network     = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["8472"]
  }
}

resource "google_compute_firewall" "teleport" {
  name    = "${var.node_tag}-teleport"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["3022-3025"]
  }
}

resource "google_compute_firewall" "planet" {
  name        = "${var.node_tag}-planet"
  description = "Planet RPC service"
  network     = "${data.google_compute_network.robotest.self_link}"

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

data "google_compute_subnetwork" "robotest" {
  name = "default"
}

# # FIXME: propagate to gravity as `--pod-cidr` and `--service-cidr`
# resource "google_compute_subnetwork" "robotest" {
#   network = "${data.google_compute_network.robotest.self_link}"
#   ip_cidr_range = "/24" # Ugh, must be a valid CIDR, how does container API work? Enumerate to pick an unoccupied block?
#   name = "${var.node_tag}-subnet"
# 
#   secondary_ip_range {
#     range_name    = "${var.node_tag}-pod-cidr"
#     ip_cidr_range = "/26"                      # enough for 64 Pods
#   }
# 
#   secondary_ip_range {
#     range_name    = "${var.node_tag}-service-cidr"
#     ip_cidr_range = "/24"
#   }
# }

