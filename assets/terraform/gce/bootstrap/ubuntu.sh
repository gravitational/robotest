#!/bin/bash
#
# File passed to VM at creation time
#
set -euo pipefail

touch /var/lib/bootstrap_started

# disable Hyper-V time sync
echo 2dd1ce17-079e-403c-b352-a1921ee207ee > /sys/bus/vmbus/drivers/hv_util/unbind

apt update 
apt install -y chrony lvm2 curl wget thin-provisioning-tools
curl https://bootstrap.pypa.io/get-pip.py | python -
pip install --upgrade awscli

mkfs.ext4 -F /dev/sdc
echo -e '/dev/sdc\t/var/lib/gravity/planet/etcd\text4\tdefaults\t0\t2' >> /etc/fstab

mkdir -p /var/lib/gravity/planet/etcd /var/lib/data
mount /var/lib/gravity/planet/etcd

chown -R 1000:1000 /var/lib/gravity /var/lib/data /var/lib/gravity/planet/etcd
sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers

# Preflight tests expectations
modprobe br_netfilter || true
modprobe overlay || true
modprobe ebtables || true
sysctl -w net.ipv4.ip_forward=1
sysctl -w net.bridge.bridge-nf-call-iptables=1

# make changes permanent
cat > /etc/sysctl.d/50-telekube.conf <<EOF
net.ipv4.ip_forward=1
net.bridge.bridge-nf-call-iptables=1
EOF
cat > /etc/modules-load.d/telekube.conf <<EOF
br_netfilter
overlay
ebtables
EOF

# robotest might SSH before bootstrap script is complete (and will fail)
touch /var/lib/bootstrap_complete
