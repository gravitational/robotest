#!/bin/bash
#
# VM bootstrap script for Debian/Ubuntu
#
set -euo pipefail

touch /var/lib/bootstrap_started

apt update
apt install -y chrony lvm2 curl wget thin-provisioning-tools
curl https://bootstrap.pypa.io/get-pip.py | python -
pip install --upgrade awscli

mkfs.ext4 -F /dev/sdc
echo -e '/dev/sdc\t/var/lib/gravity/planet/etcd\text4\tdefaults\t0\t2' >> /etc/fstab

mkdir -p /var/lib/gravity/planet/etcd /var/lib/data
mount /var/lib/gravity/planet/etcd

chown -R $${service_uid}:$${service_gid} /var/lib/gravity /var/lib/data /var/lib/gravity/planet/etcd
sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers

# Required kernel modules
modprobe br_netfilter || true
modprobe overlay || true
modprobe ebtable_filter || true

# Make changes permanent
cat > /etc/sysctl.d/50-telekube.conf <<EOF
net.ipv4.ip_forward=1
net.bridge.bridge-nf-call-iptables=1
EOF
cat > /etc/modules-load.d/telekube.conf <<EOF
br_netfilter
overlay
ebtables
EOF
sysctl -p /etc/sysctl.d/50-telekube.conf

# This marks the bootstrap step as complete for robotest
touch /var/lib/bootstrap_complete
