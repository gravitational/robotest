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

apt update
apt install -y python-pip lvm2 curl wget
pip install --upgrade awscli

mkfs.ext4 /dev/xvdc
echo -e '/dev/xvdc\t/var/lib/gravity/planet/etcd\text4\tdefaults\t0\t2' >> /etc/fstab

mkdir -p /var/lib/gravity/planet/etcd /var/lib/data
mount /var/lib/gravity/planet/etcd

chown -R 1000:1000 /var/lib/gravity /var/lib/data /var/lib/gravity/planet/etcd
sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers

umount /dev/xvdb || true
wipefs -a /dev/xvdb || true

# robotest might SSH before bootstrap script is complete (and will fail)
touch /var/lib/bootstrap_complete
