#!/bin/bash
#
# File passed to VM at creation time
#
set -euo pipefail

apt update 
apt install -y chrony python-pip lvm2 curl wget 
pip install --upgrade awscli

mkfs.ext4 /dev/sdc
echo -e '/dev/sdc\t/var/lib/gravity/planet/etcd\text4\tdefaults\t0\t2' >> /etc/fstab

mkdir -p /var/lib/gravity/planet/etcd /var/lib/data
mount /var/lib/gravity/planet/etcd

chown -R 1000:1000 /var/lib/gravity /var/lib/data /var/lib/gravity/planet/etcd
sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers

# robotest might SSH before bootstrap script is complete (and will fail)
touch /var/lib/bootstrap_complete