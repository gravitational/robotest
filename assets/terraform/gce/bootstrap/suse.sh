#!/bin/bash
#
# VM bootstrap script for SuSE
#
set -exuo pipefail

touch /var/lib/bootstrap_started

curl https://bootstrap.pypa.io/get-pip.py | python -
pip install --upgrade awscli

mkdir -p /var/lib/gravity/planet/etcd /var/lib/data

etcd_device=sdc
etcd_dir=/var/lib/gravity/planet/etcd
if ! grep -qs "$etcd_dir" /proc/mounts; then
  mkfs.ext4 -F /dev/sdc
  sed -i.bak "/$etcd_device/d" /etc/fstab
  echo -e "/dev/$etcd_device\t$etcd_dir\text4\tdefaults\t0\t2" >> /etc/fstab
  mount $etcd_dir
fi

real_user=$(logname)
service_uid=$(id $real_user -u)
service_gid=$(id $real_user -g)

chown -R $service_uid:$service_gid /var/lib/gravity /var/lib/data $etcd_dir
sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers

# Load required kernel modules
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

# Mark bootstrap step complete for robotest
touch /var/lib/bootstrap_complete
