#!/bin/bash
#
# VM bootstrap script for CentOS/RHEL
#
set -exuo pipefail

DIR="$(cd "$(dirname "$${BASH_SOURCE[0]}")" >/dev/null && pwd)"

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

touch /var/lib/bootstrap_started

dns_running=0
systemctl is-active --quiet dnsmasq || dns_running=$?
if [ $dns_running -eq 0 ] ; then
  systemctl stop dnsmasq || true
  systemctl disable dnsmasq
fi

if [[ $(source /etc/os-release ; echo $VERSION_ID ) == "7.2" ]] ; then
  yum install -y yum-plugin-versionlock
  yum versionlock \
        lvm2-2.02.166-1.el7_3.4.x86_64 \
        device-mapper-persistent-data-0.6.3-1.el7.x86_64 \
        device-mapper-event-libs-1.02.135-1.el7_3.4.x86_64 \
        device-mapper-event-7:1.02.135-1.el7_3.4.x86_64 \
        device-mapper-libs-7:1.02.135-1.el7_3.4.x86_64 \
        device-mapper-7:1.02.135-1.el7_3.4.x86_64
fi

yum install -y chrony python unzip lvm2 device-mapper-persistent-data

if ! aws --version; then
  curl "https://s3.amazonaws.com/aws-cli/awscli-bundle.zip" -o "awscli-bundle.zip"
  unzip awscli-bundle.zip
  ./awscli-bundle/install -i /usr/local/aws -b /usr/bin/aws
fi

etcd_dir=/var/lib/gravity/planet/etcd
if ! grep -qs "$etcd_dir" /proc/mounts; then
  etcd_device=$(get_empty_device)
  [ ! -z "$etcd_device" ] || (>&2 echo no suitable device for etcd; exit 1)

  mkfs.ext4 -F /dev/$etcd_device
  sed -i.bak "/$etcd_device/d" /etc/fstab
  echo -e "/dev/$etcd_device\t$etcd_dir\text4\tdefaults\t0\t2" >> /etc/fstab

  mkdir -p $etcd_dir /var/lib/data
  mount $etcd_dir
fi

docker_device=$(get_empty_device)
[ ! -z "$docker_device" ] || (>&2 echo no suitable device for docker; exit 1)
echo "DOCKER_DEVICE=/dev/$docker_device" > /tmp/gravity_environment

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
