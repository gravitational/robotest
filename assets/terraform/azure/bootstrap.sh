#!/bin/bash
#
# File passed to VM at creation time
#
set -euo pipefail

pvcreate -ff /dev/sdc
vgcreate data /dev/sdc
lvcreate -n var_lib_gravity -l 20%VG data
lvcreate -n var_lib_data -l 20%VG data
lvcreate -n docker -l 30%VG data
lvcreate -n etcd -l 30%VG data

sed -i.bak '/dev\/data/d' /etc/fstab
mkfs.ext4 /dev/data/var_lib_gravity
mkfs.ext4 /dev/data/var_lib_data
mkfs.ext4 /dev/data/etcd

echo -e '/dev/data/var_lib_gravity\t/var/lib/gravity\text4\tdefaults\t0\t2' >> /etc/fstab
echo -e '/dev/data/var_lib_data\t/var/lib/data\text4\tdefaults\t0\t2' >> /etc/fstab
echo -e '/dev/data/etcd\t/var/lib/gravity/planet/etcd\text4\tdefaults\t0\t2' >> /etc/fstab

mkdir -p /var/lib/gravity /var/lib/data
mount /var/lib/gravity
mount /var/lib/data
mkdir -p /var/lib/gravity/planet/etcd
mount /var/lib/gravity/planet/etcd

chown -R 1000:1000 /var/lib/gravity /var/lib/data /var/lib/gravity/planet/etcd
sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers