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

#
# system configs
#
echo "DefaultTasksMax=infinity" >> /etc/systemd/system.conf
systemctl daemon-reexec

modprobe br_netfilter || true
modprobe overlay || true
sysctl -w net.bridge.bridge-nf-call-iptables=1

cat > /usr/lib/sysctl.d/60-telekube.conf <<EOF
net.bridge.bridge-nf-call-iptables=1
EOF
cat > /etc/modules-load.d/telekube.conf <<EOF
br_netfilter
overlay
EOF

timesync_bus_name=$(get_timesync_bus_name)
if [ ! -z "$timesync_bus_name" ]; then
  # disable Hyper-V host time sync 
  echo $timesync_bus_name > /sys/bus/vmbus/drivers/hv_util/unbind
fi

#
# packages 
#

zypper --no-color --non-interactive install chrony python unzip lvm2
systemctl start chronyd

curl "https://s3.amazonaws.com/aws-cli/awscli-bundle.zip" -o "awscli-bundle.zip"
unzip awscli-bundle.zip
./awscli-bundle/install -i /usr/local/aws -b /usr/bin/aws

#
# storage
#
mount

etcd_device=$(get_empty_device)
[ ! -z "$etcd_device" ] || (>&2 echo no suitable device for etcd; exit 1)

mkfs.ext4 -F /dev/$etcd_device
echo -e "/dev/${etcd_device}\t/var/lib/gravity/planet/etcd\text4\tdefaults\t0\t2" >> /etc/fstab

mkdir -p /var/lib/gravity/planet/etcd /var/lib/data
mount /var/lib/gravity/planet/etcd

chown -R 1000:1000 /var/lib/gravity /var/lib/data /var/lib/gravity/planet/etcd

sync
docker_device=$(get_empty_device)
[ ! -z "$docker_device" ] || (>&2 echo no suitable device for docker; exit 1)
echo "DOCKER_DEVICE=/dev/$docker_device" > /tmp/gravity_environment

#
#
#
touch /var/lib/bootstrap_complete
