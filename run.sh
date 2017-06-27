#!/bin/sh

TMPDIR=/tmp/rt_$(date '+%m%d-%H%M')
mkdir -p ${TMPDIR}

P=$(PWD)
export REPORT_FILE=$(date '+%m%d-%H%M')

export DEPLOY_ACCESS_KEY=AKIAIVP3PZZMRWROJDXQ
export DEPLOY_SECRET_KEY=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
export DEPLOY_REGION=us-east-1

export CLOUD_CONFIG="
installer_url: s3://s3.gravitational.io/denis/rsenabled-telekube-3.56.3-installer.tar
script_path: /robotest/terraform/azure
state_dir: /robotest/state
cloud: azure
aws:
  access_key: ${DEPLOY_ACCESS_KEY}
  secret_key: ${DEPLOY_SECRET_KEY}
  ssh_user: ubuntu
  key_path: /robotest/config/ops.pem
  key_pair: ops
  region: ${DEPLOY_REGION}
  vpc: Create New
  docker_device: /dev/xvdb
azure: 
  subscription_id: 060a97ea-3a57-4218-9be5-dba3f19ff2b5
  client_id: bd5ef175-c52d-4352-ab1b-1e40ac8ad805
  client_secret: KF4YLEyO0C6i4fnQNO6SGjFMEGas46FKOLrpybh2/Ww=
  tenant_id: ff882432-09b0-437b-bd22-ca13c0037ded
  vm_type: Standard_F4s
  location: westus
  ssh_user: robotest
  key_path: /robotest/config/ops.pem
  authorized_keys_path: /robotest/config/ops_rsa.pub
  docker_device: /dev/sdd
"

docker run \
	-v ${P}/wd_suite/state:/robotest/state \
	-v ${P}/wd_suite/config:/robotest/config \
	-v ${P}/wd_suite/telekube-app.tar.gz:/robotest/telekube-app.tar.gz \
	-v ${P}/build/robotest-suite:/usr/bin/robotest-suite \
	-v ${P}/assets/terraform:/robotest/terraform \
	quay.io/gravitational/robotest-suite:1.0.60 \
	robotest-suite -test.timeout=48h -test.v \
	-test.parallel=10 -repeat=10 -fail-fast=false \
	-provision="$CLOUD_CONFIG" \
	-resourcegroup-file=/robotest/state/alloc.txt \
	-destroy-on-success=true -destroy-on-failure=true -always-collect-logs=true \
	-tag=rs -suite=sanity -os=ubuntu -storage-driver=devicemapper \
	'install={"flavor":"three", "nodes":3, "remote_support":false}' \
	2>&1 | tee ${P}/wd_suite/state/${REPORT_FILE}.txt

echo REPORT IN ${P}/wd_suite/state/${REPORT_FILE}.txt
