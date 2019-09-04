#
# Virtual Machine node
#

# Use locally pre-fetched image
resource "libvirt_volume" "os-qcow2" {
  name    = "os-qcow2"
  pool    = "default"
  source  = "/var/lib/libvirt/images/${var.image_name}"
  format  = "qcow2"
}

# Create a network for our VMs
resource "libvirt_network" "vm_network" {
   name       = "vm_network"
   addresses  = ["172.28.128.0/24"]
}

# Create main disk
resource "libvirt_volume" "gravity" {
  name            = "gravity-disk-${count.index}.qcow2"
  base_volume_id  = libvirt_volume.os-qcow2.id
  pool            = "default"
  size            = "${var.disk_size}"
  count           = "${var.nodes}"
}

# Use CloudInit to add our ssh-key to the instance
resource "libvirt_cloudinit_disk" "commoninit" {
  name           = "commoninit.iso"
  user_data = <<EOF
    #cloud-config
    packages: [python, curl, htop, iotop, lsof, ltrace, mc, net-tools, strace, tcpdump, telnet, vim, wget, ntp, traceroute, bash-completion]
    ssh_authorized_keys: ["${file(var.ssh_pub_key_path)}"]
    write_files:
    - content: "br_netfilter"
      path: /etc/modules-load.d/br_netfilter.conf
    - content: "ebtables"
      path: /etc/modules-load.d/ebtables.conf
    - content: "overlay"
      path: /etc/modules-load.d/overlay.conf
    - content: |
        ip_tables
        iptable_nat
        iptable_filter
      path: /etc/modules-load.d/iptables.conf
    - content: |
        net.bridge.bridge-nf-call-arptables=1
        net.bridge.bridge-nf-call-ip6tables=1
        net.bridge.bridge-nf-call-iptables=1
      path: /etc/sysctl.d/10-br-netfilter.conf
    - content: |
        net.ipv4.ip_forward=1
      path: /etc/sysctl.d/10-ipv4-forwarding-on.conf
    - content: |
        fs.may_detach_mounts=1
      path: /etc/sysctl.d/10-fs-may-detach-mounts.conf
    bootcmd:
    - echo 127.0.1.1 "${var.ssh_user}" >> /etc/hosts
    runcmd:
    - 'modprobe overlay'
    - 'modprobe br_netfilter'
    - 'modprobe ebtables'
    - 'modprobe ip_tables'
    - 'modprobe iptable_nat'
    - 'modprobe iptable_filter'
    - 'sysctl -p /etc/sysctl.d/10-br-netfilter.conf'
    - 'sysctl -p /etc/sysctl.d/10-ipv4-forwarding-on.conf'
    - 'sysctl -p /etc/sysctl.d/10-fs-may-detach-mounts.conf'
    EOF
}

# Create the machine
resource "libvirt_domain" "domain-gravity" {
  name      = "gravity${count.index}"
  memory    = "${var.memory_size}"
  vcpu      = "${var.cpu_count}"
  count     = "${var.nodes}"
  cloudinit = "${libvirt_cloudinit_disk.commoninit.id}"

  network_interface {
    hostname        = "gravity${count.index}"
    network_id      = "${libvirt_network.vm_network.id}"
    addresses       = ["172.28.128.${count.index+3}"]
    mac             = "6E:02:C0:21:62:5${count.index+3}"
    wait_for_lease  = true
  }

  # IMPORTANT
  # Ubuntu can hang if an isa-serial is not present at boot time.
  # If you find your CPU 100% and never is available this is why
  console {
    type        = "pty"
    target_port = "0"
    target_type = "serial"
  }

  console {
    type        = "pty"
    target_type = "virtio"
    target_port = "1"
  }

  disk {
    volume_id = "${element(libvirt_volume.gravity.*.id, count.index)}"
  }
}
