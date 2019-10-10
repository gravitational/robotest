#!/bin/bash
#
# VM bootstrap script for Debian/Ubuntu
#
set -exuo pipefail

etcd_device_name=sdb
etcd_dir=/var/lib/gravity/planet/etcd
DIR="$(cd "$(dirname "$${BASH_SOURCE[0]}")" >/dev/null && pwd)"

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

  chown -R $service_uid:$service_gid /var/lib/gravity $etcd_dir /home/${os_user}
  sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers
}

touch /var/lib/bootstrap_started

mkdir -p $etcd_dir /var/lib/data

remove-sshguard
secure-ssh
setup-user

# Bump number of retries for download failures
echo "APT::Acquire::Retries \"10\";" > /etc/apt/apt.conf.d/80-retries

apt-get update
apt-get install -y chrony lvm2 curl wget thin-provisioning-tools python

curl https://bootstrap.pypa.io/get-pip.py | python -
pip install --upgrade awscli

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

# Mark bootstrap step complete for robotest
touch /var/lib/bootstrap_complete
