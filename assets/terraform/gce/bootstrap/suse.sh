#!/bin/bash
#
# VM bootstrap script for SuSE
#
set -exuo pipefail

DIR="$(cd "$(dirname "$${BASH_SOURCE[0]}")" >/dev/null && pwd)"

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

## Setup modules / sysctls
# Load required kernel modules
modules="br_netfilter overlay ebtable_filter ip_tables iptable_filter iptable_nat"
for module in $modules; do
  modprobe $module || true
done
# Store the modules into a file so that the modules will be auto-reloaded
# on reboot
echo '' > /etc/modules-load.d/telekube.conf
for module in $modules; do
  echo $module >> /etc/modules-load.d/telekube.conf
done

# Make changes permanent
cat > /etc/sysctl.d/50-telekube.conf <<EOF
net.ipv4.ip_forward=1
net.bridge.bridge-nf-call-iptables=1
EOF
sysctl -p /etc/sysctl.d/50-telekube.conf

real_user=${os_user}
service_uid=$(id $real_user -u)
service_gid=$(id $real_user -g)
chown -R $service_uid:$service_gid /var/lib/gravity /var/lib/gravity/planet/etcd

sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers

# Mark bootstrap step complete for robotest
touch /var/lib/bootstrap_complete
