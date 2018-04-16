# 
# NETWORK
# 

resource "google_compute_firewall" "virtual_network" {
    name = "allow_virtual_network"
    network = "${google_compute_network.default.name}"
    priority = 200
    allow {
        protocol = "tcp"
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "ssh" {
    name = "ssh"
    network = "${google_compute_network.default.name}"
    priority = 1010
    allow {
        protocol = "tcp"
        ports = ["22"]
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "web" {
    name = "web_admin"
    network = "${google_compute_network.default.name}"
    priority = 1011
    allow {
        protocol = "tcp"
        ports = ["32009"]
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "telekube_installer" {
    name = "installer"
    network = "${google_compute_network.default.name}"
    priority = 1020
    allow {
        protocol = "tcp"
        ports = ["61008-61010", "61022-61024", "3008-3010", "6060"]
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "bandwidth" {
    name = "installer"
    network = "${google_compute_network.default.name}"
    priority = 1025
    allow {
        protocol = "tcp"
        ports = ["4242"]
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "kubelet" {
    name = "installer"
    network = "${google_compute_network.default.name}"
    priority = 1020
    allow {
        protocol = "tcp"
        ports = ["10249", "10250", "10255"]
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "serf" {
    name = "serf_peer_network"
    network = "${google_compute_network.default.name}"
    priority = 1030
    allow {
        protocol = "tcp"
        ports = ["7496", "7373"]
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "k8s_api" {
    name = "kubernetes_api_server"
    network = "${google_compute_network.default.name}"
    priority = 1040
    allow {
        protocol = "tcp"
        ports = ["8080", "6443"]
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "k8s" {
    name = "kubernetes_internal_services"
    network = "${google_compute_network.default.name}"
    priority = 1045
    allow {
        protocol = "tcp"
        ports = ["30000-32767"]
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "etcd" {
    name = "etcd"
    network = "${google_compute_network.default.name}"
    priority = 1050
    allow {
        protocol = "tcp"
        ports = ["2379", "2380", "4001", "7001"]
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "registry" {
    name = "docker_registry"
    network = "${google_compute_network.default.name}"
    priority = 1060
    allow {
        protocol = "tcp"
        ports = ["5000", "2380", "4001", "7001"]
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "docker_registry" {
    name = "serf_peer_network"
    network = "${google_compute_network.default.name}"
    priority = 1070
    allow {
        protocol = "tcp"
        ports = ["5000"]
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "overlay_network" {
    name = "overlay_network"
    network = "${google_compute_network.default.name}"
    priority = 1080
    allow {
        protocol = "tcp"
        ports = ["8472"]
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "teleport" {
    name = "teleport_services"
    network = "${google_compute_network.default.name}"
    priority = 1090
    allow {
        protocol = "tcp"
        ports = ["3022-3025"]
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "planet" {
    name = "planet_rpc"
    network = "${google_compute_network.default.name}"
    priority = 1100
    allow {
        protocol = "tcp"
        ports = ["7575"]
    }

    source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "ntp" {
    name = "planet_rpc"
    network = "${google_compute_network.default.name}"
    priority = 1120
    allow {
        protocol = "udp"
        ports = ["123"]
    }

    source_ranges = ["0.0.0.0/0"]
}

# FIXME: left-overs Azure network configuration

resource "azurerm_network_security_group" "robotest" {
  name                = "robotest"
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.robotest.name}"
}

resource "azurerm_virtual_network" "robotest" {
  name                = "robotest"
  address_space       = ["10.40.0.0/16"]
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.robotest.name}"
}

resource "azurerm_subnet" "robotest_a" {
  name                      = "robotest_10_40_2_0"
  resource_group_name       = "${azurerm_resource_group.robotest.name}"
  virtual_network_name      = "${azurerm_virtual_network.robotest.name}"
  address_prefix            = "10.40.2.0/24"
  network_security_group_id = "${azurerm_network_security_group.robotest.id}"
}

