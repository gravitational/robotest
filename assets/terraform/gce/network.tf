#
# Network
#

# resource "google_compute_firewall" "robotest" {
#   name        = "${var.node_tag}-fw"
#   description = "Robotest firewall rules"
#   network     = "${data.google_compute_network.robotest.self_link}"
# 
#   target_tags = ["${var.node_tag}"]
# 
#   enable_logging = "true"
# 
#   allow {
#     protocol = "icmp"
#   }
# 
#   # SSH access
#   allow {
#     protocol = "tcp"
# 
#     ports = ["22", "61822"]
#   }
# 
#   # VXLAN
#   allow {
#     protocol = "udp"
# 
#     ports = ["4789"]
#   }
# 
#   # Web UI
#   allow {
#     protocol = "tcp"
#     ports    = ["32009"]
#   }
# 
#   # Internal installer ports
#   allow {
#     protocol = "tcp"
#     ports    = ["61008-61010", "61022-61024", "3008-3010", "3012", "6060"]
#   }
# 
#   # Bandwidth network test
#   allow {
#     protocol = "tcp"
#     ports    = ["4242"]
#   }
# 
#   # Kubelet
#   allow {
#     protocol = "tcp"
#     ports    = ["10248", "10250"]
#   }
# 
#   allow {
#     protocol = "tcp"
#     ports    = ["7496", "7373"]
#   }
# 
#   # Kubernetes API server and internal services
#   allow {
#     protocol = "tcp"
#     ports    = ["6443", "30000-32767"]
#   }
# 
#   # Internal etcd cluster
#   allow {
#     protocol = "tcp"
#     ports    = ["2379", "2380", "4001", "7001"]
#   }
# 
#   # Internal docker registry
#   allow {
#     protocol = "tcp"
#     ports    = ["5000"]
#   }
# 
#   # Flannel overlay network
#   allow {
#     protocol = "tcp"
#     ports    = ["8472"]
#   }
# 
#   # Teleport
#   allow {
#     protocol = "tcp"
#     ports    = ["3022-3025"]
#   }
# 
#   # Planet RPC service
#   allow {
#     protocol = "tcp"
#     ports    = ["7575"]
#   }
# 
#   # NTP
#   allow {
#     protocol = "udp"
#     ports    = ["123"]
#   }
# }

data "google_compute_network" "robotest" {
  name = "${var.network}"
}

data "google_compute_subnetwork" "robotest" {
  name = "${var.subnet}"
}

# # TODO: propagate to gravity as `--pod-cidr` and `--service-cidr`
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
