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
# File passed to VM at creation time
#
set -euo pipefail

function add_host {
  local hostname=$(hostname)
  if ! grep -q "$hostname" /etc/hosts; then
    echo -e "127.0.0.1\t$hostname" >> /etc/hosts
  fi
}

touch /var/lib/bootstrap_started

# disable Hyper-V time sync
echo 2dd1ce17-079e-403c-b352-a1921ee207ee > /sys/bus/vmbus/drivers/hv_util/unbind

apt update
apt install -y chrony lvm2 curl wget thin-provisioning-tools
curl https://bootstrap.pypa.io/get-pip.py | python -
pip install --upgrade awscli

mkfs.ext4 -F /dev/sdc
echo -e '/dev/sdc\t/var/lib/gravity/planet/etcd\text4\tdefaults\t0\t2' >> /etc/fstab

mkdir -p /var/lib/gravity/planet/etcd /var/lib/data
mount /var/lib/gravity/planet/etcd

chown -R 1000:1000 /var/lib/gravity /var/lib/data /var/lib/gravity/planet/etcd
sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers

# Preflight tests expectations
modprobe br_netfilter || true
modprobe overlay || true
modprobe ebtables || true
modprobe ip_tables || true
modprobe iptable_filter || true
modprobe iptable_nat || true
sysctl -w net.ipv4.ip_forward=1
sysctl -w net.bridge.bridge-nf-call-iptables=1

# make changes permanent
cat > /etc/sysctl.d/50-telekube.conf <<EOF
net.ipv4.ip_forward=1
net.bridge.bridge-nf-call-iptables=1
EOF
cat > /etc/modules-load.d/telekube.conf <<EOF
br_netfilter
overlay
ebtables
ip_tables
iptable_filter
iptable_nat
EOF

add_host

# robotest might SSH before bootstrap script is complete (and will fail)
touch /var/lib/bootstrap_complete
