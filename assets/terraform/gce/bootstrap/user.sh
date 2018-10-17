#!/bin/bash
set -eu

real_user=$1
service_uid=$(id $real_user -u)
service_gid=$(id $real_user -g)
chown -R $service_uid:$service_gid /var/lib/gravity /var/lib/gravity/planet/etcd

sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers
