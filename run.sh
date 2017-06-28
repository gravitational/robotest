#!/bin/sh

#
# could be local .tar installer or s3:// or http(s) URL
#
export INSTALLER_URL=

# multiple tests are supported
# OS could be ubuntu,centos,rhel 
# storage driver: could be devicemapper,loopback,overlay,overlay2 
#  separate multiple values by comma for OS and storage driver
#  tests need be separated by space
export RUN_TEST='install={"flavor":"three", "nodes":3, "remote_support":false}'
export TEST_OS=ubuntu
export STORAGE_DRIVER=devicemapper

export REPEAT_TESTS=1
export PARALLEL_TESTS=1
export FAIL_FAST=false

# choose something relatively unique to avoid intersection with other people runs
# tag would prefix cloud resource groups for your test runs
export TAG=$(id -run)

# what should happen with provisioned VMs on individual test success or failure
export DESTROY_ON_SUCCESS=true
export DESTROY_ON_FAILURE=true

# 
# SSH keys to configure remote hosts
# 
export SSH_KEY=
export SSH_PUB=

# which cloud to use : aws or azure
# define rest of the keys in the CLOUD_CONFIG below
export DEPLOY_TO=azure

export CLOUD_CONFIG="
installer_url: ${INSTALLER_URL}
script_path: /robotest/terraform/${DEPLOY_TO}
state_dir: /robotest/state
cloud: ${DEPLOY_TO}
aws:
  access_key: **************
  secret_key: **************
  ssh_user: ubuntu
  key_path: /robotest/config/ops.pem
  key_pair: **************
  region: **************
  vpc: Create New
  docker_device: /dev/xvdb
azure: 
  subscription_id: **************
  client_id: **************
  client_secret: **************
  tenant_id: **************
  vm_type: Standard_F4s
  location: westus
  ssh_user: robotest
  key_path: /robotest/config/ops.pem
  authorized_keys_path: /robotest/config/ops_rsa.pub
  docker_device: /dev/sdd
"

#
# you generally don't need to change anything beyond this line
# 

P=$(PWD)
export REPORT_FILE=$(date '+%m%d-%H%M')
mkdir -p ${P}/wd_suite/state/${TAG}

docker run \
	-v ${P}/wd_suite/state:/robotest/state \
	-v ${SSH_KEY}:/robotest/config/ops.pem \
  -v ${SSH_PUB}:/robotest/config/ops_rsa.pub \
	quay.io/gravitational/robotest-suite:latest \
	robotest-suite -test.timeout=48h -test.v \
	-test.parallel=${PARALLEL_TESTS} -repeat=${REPEAT_TESTS} -fail-fast=false \
	-provision="$CLOUD_CONFIG" \
	-resourcegroup-file=/robotest/state/alloc.txt \
	-destroy-on-success=${DESTROY_ON_SUCCESS} -destroy-on-failure=${DESTROY_ON_FAILURE} -always-collect-logs=true \
	-tag=${TAG} -suite=sanity -os=${TEST_OS} -storage-driver=${STORAGE_DRIVER} \
	${RUN_TEST} \
	2>&1 | tee ${P}/wd_suite/state/${TAG}/${REPORT_FILE}.txt

echo ${P}/wd_suite/state/${REPORT_FILE}.txt
