#!/bin/bash
#
# VM bootstrap script for Debian/Ubuntu
#
set -exuo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null && pwd)"

touch /var/lib/bootstrap_started

apt update
apt install -y chrony lvm2 curl wget thin-provisioning-tools
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

source $(DIR)/modules.sh

# Mark bootstrap step complete for robotest
touch /var/lib/bootstrap_complete
