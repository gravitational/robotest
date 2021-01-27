#!/bin/bash
#
# Copyright 2020 Gravitational, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#
# VM bootstrap script for Debian/Ubuntu
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
  systemctl reload ssh
}

function remove-sshguard {
  if systemctl is-active --quiet sshguard; then
    apt-get -y remove --auto-remove sshguard
    apt-get -y purge --auto-remove sshguard
  fi
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
    (useradd --system --non-unique --gid $service_gid --uid $service_uid ${os_user} 2>/dev/null; err=$?; if (( $err != 9 )); then exit $err; fi) || true
  fi

  if [ ! -d "/home/${os_user}/.ssh" ]; then
    mkdir -p /home/${os_user}/.ssh
    echo "${ssh_pub_key}" | tee /home/${os_user}/.ssh/authorized_keys
    chmod 0700 /home/${os_user}/.ssh
    chmod 0600 /home/${os_user}/.ssh/authorized_keys
    chsh -s /bin/bash ${os_user}
  fi

  chown -R $service_uid:$service_gid /home/${os_user}
  sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers
}

mkdir -p $etcd_dir /var/lib/data

remove-sshguard
secure-ssh
setup-user

# Bump number of retries for download failures
echo "APT::Acquire::Retries \"10\";" > /etc/apt/apt.conf.d/80-retries

apt-get update
apt-get install -y chrony lvm2 curl wget thin-provisioning-tools

if ! /usr/local/bin/aws --version; then
  apt-get install -y unzip
  curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64-2.0.30.zip" -o "awscliv2.zip"
  unzip awscliv2.zip
  ./aws/install
fi

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
net.ipv4.ip_forward=1
net.bridge.bridge-nf-call-iptables=1
net.ipv4.tcp_keepalive_time=60
net.ipv4.tcp_keepalive_intvl=60
net.ipv4.tcp_keepalive_probes=5
EOF
sysctl -p /etc/sysctl.d/50-telekube.conf
