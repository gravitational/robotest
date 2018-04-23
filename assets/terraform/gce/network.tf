#
# Network
#

resource "google_compute_firewall" "ssh" {
  name    = "${var.cluster_name}-allow_ssh"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }
}

resource "google_compute_firewall" "web" {
  name    = "${var.cluster_name}-web_admin"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["32009"]
  }
}

resource "google_compute_firewall" "telekube_installer" {
  name    = "${var.cluster_name}-installer"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["61008-61010", "61022-61024", "3008-3010", "6060"]
  }
}

resource "google_compute_firewall" "bandwidth" {
  name    = "${var.cluster_name}-bandwidth_test"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["4242"]
  }
}

resource "google_compute_firewall" "kubelet" {
  name    = "${var.cluster_name}-kubelet"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["10249", "10250", "10255"]
  }
}

resource "google_compute_firewall" "serf" {
  name    = "${var.cluster_name}-serf_peer_network"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["7496", "7373"]
  }
}

resource "google_compute_firewall" "k8s_api" {
  name    = "${var.cluster_name}-kubernetes_api_server"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["8080", "6443"]
  }
}

resource "google_compute_firewall" "k8s" {
  name    = "${var.cluster_name}-kubernetes_internal_services"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["30000-32767"]
  }
}

resource "google_compute_firewall" "etcd" {
  name    = "${var.cluster_name}-etcd"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["2379", "2380", "4001", "7001"]
  }
}

resource "google_compute_firewall" "registry" {
  name    = "${var.cluster_name}-docker_registry"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["5000"]
  }
}

resource "google_compute_firewall" "overlay_network" {
  name    = "${var.cluster_name}-overlay_network"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["8472"]
  }
}

resource "google_compute_firewall" "teleport" {
  name    = "${var.cluster_name}-teleport_services"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["3022-3025"]
  }
}

resource "google_compute_firewall" "planet" {
  name    = "${var.cluster_name}-planet_rpc"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "tcp"
    ports    = ["7575"]
  }
}

resource "google_compute_firewall" "ntp" {
  name    = "${var.cluster_name}-ntp"
  network = "${data.google_compute_network.robotest.self_link}"

  allow {
    protocol = "udp"
    ports    = ["123"]
  }
}

data "google_compute_network" "robotest" {
  name = "default"
}

# resource "google_compute_network" "robotest" {
#   name = "default"
# 
#   # Each region gets a new subnet automatically
#   # If using this, the subnetwork resource is probably unnecessary
#   auto_create_subnetworks = "true"
# 
#   # TODO: or have a single robotest subnetwork (10.40.2.0/24)
#   # for all robotest clusters (either in a run or across multiple runs)
#   # auto_create_subnetworks = "false"
# }


# # FIXME: no way to make virtual private networks with the same address
# # range per group
# resource "google_compute_subnetwork" "robotest" {
#   name          = "robotest_10_40_2_0"
#   ip_cidr_range = "10.40.2.0/24"
#   network       = "${google_compute_network.robotest.self_link}"
# }

