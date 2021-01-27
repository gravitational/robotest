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
# VM bootstrap script for CentOS/RHEL
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

function retry {
  local count=0
  while ! $@; do
    ((count++)) && ((count==20)) && return 1
    sleep 5
  done
  return 0
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
    chown -R $service_uid:$service_gid /home/${os_user}
    # install semanage
    yum -y install policycoreutils-python-utils
    # FIXME: make sure that SELinux is in effect for the command below (`getenforce`)
    retry semanage fcontext -a -t user_home_t /home/${os_user}
    retry restorecon -vR /home/${os_user}
  fi

  chown -R $service_uid:$service_gid /home/${os_user}
  sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers
}

# Yum hits "Error: Cannot retrieve metalink for repository: epel." on some images.
# According to https://stackoverflow.com/a/27667111, this is a certificate issue.
yum --disablerepo=epel -y update ca-certificates
yum -y install chrony

mkdir -p $etcd_dir /var/lib/data
secure-ssh
setup-user

dns_running=0
systemctl is-active --quiet dnsmasq || dns_running=$?
if [ $dns_running -eq 0 ] ; then
  systemctl stop dnsmasq || true
  systemctl disable dnsmasq
fi

if ! /usr/local/bin/aws --version; then
  yum -y install unzip
  curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64-2.0.30.zip" -o "awscliv2.zip"
  unzip awscliv2.zip
  ./aws/install
  rm -rf awscliv2.zip ./aws
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
if sysctl -w fs.may_detach_mounts=1; then
  # fs.may_detach_mounts is needed in rhel/cent 7, but not present in rhel/cent 8
  # https://gravitational.com/gravity/docs/faq/#kubernetes-pods-stuck-in-terminating-state
  echo "fs.may_detach_mounts=1" >> /etc/sysctl.d/50-telekube.conf
fi

sysctl -p /etc/sysctl.d/50-telekube.conf
