#!/bin/bash
#
# File passed to VM at creation time
#
set -euo pipefail

touch /var/lib/bootstrap_started

systemctl stop dnsmasq
systemctl disable dnsmasq

mount

yum install -y chrony python unzip lvm2 device-mapper-persistent-data
curl "https://s3.amazonaws.com/aws-cli/awscli-bundle.zip" -o "awscli-bundle.zip"
unzip awscli-bundle.zip
./awscli-bundle/install -i /usr/local/aws -b /usr/bin/aws

mkfs.ext4 -F /dev/sdc
echo -e '/dev/sdc\t/var/lib/gravity/planet/etcd\text4\tdefaults\t0\t2' >> /etc/fstab

mkdir -p /var/lib/gravity/planet/etcd /var/lib/data
mount /var/lib/gravity/planet/etcd

chown -R 1000:1000 /var/lib/gravity /var/lib/data /var/lib/gravity/planet/etcd
sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers

# apparently centos WAAgent on azure will auto create FS and mount all data disks, need undo that
umount /dev/sdd1 || : 
wipefs -af /dev/sdd || :

# robotest might SSH before bootstrap script is complete (and will fail)
touch /var/lib/bootstrap_complete