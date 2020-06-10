#!/bin/bash
#
# VM bootstrap script for SuSE
#
set -o errexit
set -o errtrace
set -o xtrace
set -o nounset
set -o pipefail

rm -f /var/lib/bootstrap_*
touch /var/lib/bootstrap_started
trap exit_handler EXIT

etcd_device_name=sdb
etcd_dir=/var/lib/gravity/planet/etcd
DIR="$(cd "$(dirname "$${BASH_SOURCE[0]}")" >/dev/null && pwd)"

function exit_handler {
  if [[ $? -ne 0 ]]; then
    touch -f /var/lib/bootstrap_failed
  else
    touch -f /var/lib/bootstrap_complete
  fi
}

function secure-ssh {
  local sshd_config=/etc/ssh/sshd_config
  cp $sshd_config $sshd_config.old
  (grep -qE '(\#?)\WPasswordAuthentication' $sshd_config && \
    sed -re 's/^(\#?)\W*(PasswordAuthentication)([[:space:]]+)yes/\2\3no/' -i $sshd_config) || \
    echo 'PasswordAuthentication no' >> $sshd_config
  (grep -qE '(\#?)\WChallengeResponseAuthentication' $sshd_config && \
    sed -re 's/^(\#?)\W*(ChallengeResponseAuthentication)([[:space:]]+)yes/\2\3no/' -i $sshd_config) || \
    echo 'ChallengeResponseAuthentication no' >> $sshd_config
  systemctl reload sshd
}

function setup-user {
  local service_uid=$(id ${os_user} -u 2>/dev/null || true)
  local service_gid=$(id ${os_user} -g 2>/dev/null || true)

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
  fi

  chown -R $service_uid:$service_gid /home/${os_user}
  sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers
}

mkdir -p $etcd_dir /var/lib/data

secure-ssh
setup-user

curl https://bootstrap.pypa.io/get-pip.py | python -
pip install --upgrade awscli

mkdir -p /var/lib/gravity/planet/etcd /var/lib/data

if ! grep -qs "$etcd_dir" /proc/mounts; then
  mkfs.ext4 -F /dev/$etcd_device_name
  sed -i.bak "/$etcd_device_name/d" /etc/fstab
  echo -e "/dev/$etcd_device_name\t$etcd_dir\text4\tdefaults\t0\t2" >> /etc/fstab
  mount $etcd_dir
fi

# grow-root-fs expands '/' to use all available space on the device.
#
# The GCP sles-12-sp5-v20200610 image doesn't recognize extra space in drives
# by default, and limits the '/' partition to 10G without this kick.
#
# See https://cloud.google.com/compute/docs/disks/add-persistent-disk#resize_partitions
function grow-root-fs {
  echo "Filesystem utilization before grow-root-fs:"
  df -h

  local root_fs_dev=$(findmnt --noheadings -o SOURCE /) # e.g. /dev/sda3
  local root_fs_dev_name=$(lsblk --noheadings -o kname $root_fs_dev) # e.g. sda3
  local partition_number=$(cat /sys/class/block/$root_fs_dev_name/partition) # e.g. 3
  local parent_dev_name=$(lsblk --noheadings -o pkname $root_fs_dev) # e.g. sda
  local parent_dev=/dev/$parent_dev_name # e.g. /dev/sda

  growpart $parent_dev $partition_number
  xfs_growfs /

  echo "Filesystem utilization after grow-root-fs:"
  df -h
}

grow-root-fs

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
net.ipv4.tcp_keepalive_time=60
net.ipv4.tcp_keepalive_intvl=60
net.ipv4.tcp_keepalive_probes=5
EOF
sysctl -p /etc/sysctl.d/50-telekube.conf
