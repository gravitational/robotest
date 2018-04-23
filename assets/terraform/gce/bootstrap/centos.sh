#!/bin/bash
#
# VM bootstrap script for CentOS/RHEL
#
set -exuo pipefail

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

curl "https://s3.amazonaws.com/aws-cli/awscli-bundle.zip" -o "awscli-bundle.zip"
unzip awscli-bundle.zip
./awscli-bundle/install -i /usr/local/aws -b /usr/bin/aws

etcd_device=$(get_empty_device)
[ ! -z "$etcd_device" ] || (>&2 echo no suitable device for etcd; exit 1)

mkfs.ext4 -F /dev/$etcd_device
echo -e "/dev/$etcd_device\t/var/lib/gravity/planet/etcd\text4\tdefaults\t0\t2" >> /etc/fstab

mkdir -p /var/lib/gravity/planet/etcd /var/lib/data
mount /var/lib/gravity/planet/etcd

service_uid=$(id -u)
service_gid=$(id -g)

chown -R $service_uid:$service_gid /var/lib/gravity /var/lib/data /var/lib/gravity/planet/etcd
sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers

docker_device=$(get_empty_device)
[ ! -z "$docker_device" ] || (>&2 echo no suitable device for docker; exit 1)
echo "DOCKER_DEVICE=/dev/$docker_device" > /tmp/gravity_environment

# # FIXME: figure out if these are necessary
# systemctl disable firewalld || true
# systemctl stop firewalld || true
# iptables --flush --wait
# iptables --delete-chain
# iptables --table nat --flush
# iptables --table filter --flush
# iptables --table nat --delete-chain
# iptables --table filter --delete-chain

# Required kernel modules
modprobe br_netfilter || true
modprobe overlay || true
modprobe ebtable_filter || true

# Make changes permanent
cat > /etc/sysctl.d/50-telekube.conf <<EOF
net.ipv4.ip_forward=1
net.bridge.bridge-nf-call-iptables=1
EOF
if sysctl -q fs.may_detach_mounts >/dev/null 2>&1; then
  echo "fs.may_detach_mounts=1" >> /etc/sysctl.d/50-telekube.conf
fi
cat > /etc/modules-load.d/telekube.conf <<EOF
br_netfilter
overlay
ebtables
EOF
sysctl -p /etc/sysctl.d/50-telekube.conf

# Mark bootstrap step complete for robotest
touch /var/lib/bootstrap_complete
