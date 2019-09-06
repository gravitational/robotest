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
  name      = "commoninit.iso"
  user_data = "${templatefile("${path.module}/cloud_init.cfg", {
    ssh_pub_key = "${file(var.ssh_pub_key_path)}",
    ssh_user = var.ssh_user 
  })}"
}

# Create the machine
resource "libvirt_domain" "domain-gravity" {
  name      = "gravity${count.index}"
  memory    = "${var.memory}"
  vcpu      = "${var.cpu}"
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
