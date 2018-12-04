#!/bin/bash
#
# VM bootstrap script for CentOS/RHEL
#
set -exuo pipefail

DIR="$(cd "$(dirname "$${BASH_SOURCE[0]}")" >/dev/null && pwd)"

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

etcd_device_name=sdb
etcd_dir=/var/lib/gravity/planet/etcd
mkdir -p $etcd_dir /var/lib/data
if ! grep -qs "$etcd_dir" /proc/mounts; then
  mkfs.ext4 -F /dev/$etcd_device_name
  sed -i.bak "/$etcd_device_name/d" /etc/fstab
  echo -e "/dev/$etcd_device_name\t$etcd_dir\text4\tdefaults\t0\t2" >> /etc/fstab
  mount $etcd_dir
fi

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
fs.may_detach_mounts=1
net.ipv4.ip_forward=1
net.bridge.bridge-nf-call-iptables=1
net.ipv4.tcp_keepalive_time=60
net.ipv4.tcp_keepalive_intvl=60
net.ipv4.tcp_keepalive_probes=5
EOF
sysctl -p /etc/sysctl.d/50-telekube.conf

service_uid=$(id ${os_user} -u 2>/dev/null || true)
service_gid=$(id ${os_user} -g 2>/dev/null || true)

if [ -z "$service_gid" ]; then
  service_gid=1000
  (groupadd --system --non-unique --gid $service_gid ${os_user} 2>/dev/null; err=$?; if (( $err != 9 )); then exit $err; fi) || true
fi

if [ -z "$service_uid" ]; then
  service_uid=1000
  useradd --system --non-unique -g $service_gid -u $service_uid ${os_user}
fi

if [ ! -d "/home/${os_user}/.ssh" ]; then
  mkdir -p /home/${os_user}/.ssh
  echo "${ssh_pub_key}" | tee /home/${os_user}/.ssh/authorized_keys
  chmod 0700 /home/${os_user}/.ssh
  chmod 0600 /home/${os_user}/.ssh/authorized_keys
  chown -R $service_uid:$service_gid /home/${os_user}
  # FIXME: make sure that SELinux is in effect for the command below (`getenforce`)
  semanage fcontext -a -t user_home_t /home/${os_user}
  restorecon -vR /home/${os_user}
fi

chown -R $service_uid:$service_gid /var/lib/gravity /var/lib/gravity/planet/etcd
sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers

# Mark bootstrap step complete for robotest
touch /var/lib/bootstrap_complete
