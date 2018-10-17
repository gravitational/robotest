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

source $(DIR)/modules.sh
source $(DIR)/user.sh ${os_user}

# Mark bootstrap step complete for robotest
touch /var/lib/bootstrap_complete
