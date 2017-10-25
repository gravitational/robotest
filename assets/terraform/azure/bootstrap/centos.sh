#!/bin/bash
#
# File passed to VM at creation time
#
set -euo pipefail

function devices {
    lsblk --raw --noheadings -I 8,9,202,252,253,259 $@
}

function get_empty_device {
    for device in $(devices --output=NAME); do
        local type=$(devices --output=TYPE /dev/$device)
        if [ "$type" != "part" ] && [[ -z "$(devices --output=FSTYPE /dev/$device)" ]]; then
            echo $device
            exit 0
        fi
    done
}

function get_vmbus_attr {
  local dev_path=$1
  local attr=$2

  cat $dev_path/$attr | head -n1
}

function get_timesync_bus_name {
  local timesync_bus_id='{9527e630-d0ae-497b-adce-e80ab0175caf}'
  local vmbus_sys_path='/sys/bus/vmbus/devices'

  for device in $vmbus_sys_path/*; do
    local id=$(get_vmbus_attr $device "id")
    local class_id=$(get_vmbus_attr $device "class_id")
    if [ "$class_id" == "$timesync_bus_id" ]; then
      echo $(basename $device); exit 0
    fi
  done
}

touch /var/lib/bootstrap_started

timesync_bus_name=$(get_timesync_bus_name)
if [ ! -z "$timesync_bus_name" ]; then
  # disable Hyper-V host time sync 
  echo $timesync_bus_name > /sys/bus/vmbus/drivers/hv_util/unbind
fi

dnsrunning=0
systemctl is-active --quiet dnsmasq || dnsrunning=$?
if [ $dnsrunning -eq 0 ] ; then 
  systemctl stop dnsmasq || true
  systemctl disable dnsmasq 
fi

mount

yum install -y chrony python unzip lvm2 device-mapper-persistent-data
curl "https://s3.amazonaws.com/aws-cli/awscli-bundle.zip" -o "awscli-bundle.zip"
unzip awscli-bundle.zip
./awscli-bundle/install -i /usr/local/aws -b /usr/bin/aws

etcd_device=$(get_empty_device)
[ ! -z "$etcd_device" ] || (>&2 echo no suitable device for etcd; exit 1)

mkfs.ext4 -F /dev/$etcd_device
echo -e "/dev/${etcd_device}\t/var/lib/gravity/planet/etcd\text4\tdefaults\t0\t2" >> /etc/fstab

mkdir -p /var/lib/gravity/planet/etcd /var/lib/data
mount /var/lib/gravity/planet/etcd

chown -R 1000:1000 /var/lib/gravity /var/lib/data /var/lib/gravity/planet/etcd
sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers

sync
docker_device=$(get_empty_device)
[ ! -z "$docker_device" ] || (>&2 echo no suitable device for docker; exit 1)
echo "DOCKER_DEVICE=/dev/$docker_device" > /tmp/gravity_environment

if [[ $(awk -F'=' '/VERSION_ID/ {print $2}' /etc/os-release) == '"7.3"' ]] ; then
  systemctl enable firewalld
  systemctl start firewalld
  #
  # configure firewall rules
  # 
  firewall-cmd --zone=trusted --add-source=10.244.0.0/16 --permanent # pod subnet
  firewall-cmd --zone=trusted --add-source=10.100.0.0/16 --permanent # service subnet
  firewall-cmd --zone=trusted --add-interface=eth0 --permanent       # enable eth0 in trusted zone so nodes can communicate
  firewall-cmd --zone=trusted --add-masquerade --permanent           # masquerading so packets can be routed back
  firewall-cmd --reload
  systemctl restart firewalld
fi

# robotest might SSH before bootstrap script is complete (and will fail)
touch /var/lib/bootstrap_complete
