#!/bin/bash
#
# File passed to VM at creation time
#
set -euo pipefail

function devices {
    lsblk --raw --noheadings -I 8,9,202,252,253,259 $@
}

function get_docker_device {
    for device in $(devices --output=NAME); do
        local partitions=$(devices --output=NAME /dev/$device | wc -l)
        if (( $partitions==1 )) && [[ -z "$(devices --output=FSTYPE /dev/$device)" ]]; then
            echo $device
            exit 0
        fi
    done
}

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

docker_device=$(get_docker_device)
[ ! -z "$docker_device" ] || (>&2 echo no suitable device for docker; exit 1)
echo "/dev/$docker_device" > ~/.gravity_docker_device

#
# configure firewall rules
# 
firewall-cmd --zone=trusted --add-source=10.244.0.0/16 --permanent # pod subnet
firewall-cmd --zone=trusted --add-source=10.100.0.0/16 --permanent # service subnet
firewall-cmd --zone=trusted --add-interface=eth0 --permanent       # enable eth0 in trusted zone so nodes can communicate
firewall-cmd --zone=trusted --add-masquerade --permanent           # masquerading so packets can be routed back
firewall-cmd --reload
systemctl restart firewalld

# robotest might SSH before bootstrap script is complete (and will fail)
touch /var/lib/bootstrap_complete
