#!/bin/bash +x

#
# could be local .tar installer or s3:// or http(s) URL
#
INSTALLER_URL=${INSTALLER_URL:-$1}

# OS could be ubuntu,centos,rhel 
# storage driver: could be devicemapper,loopback,overlay,overlay2 
#  separate multiple values by comma for OS and storage driver
RUN_TEST=${RUN_TEST:-$2}

TEST_OS=${TEST_OS:-ubuntu}
STORAGE_DRIVER=${STORAGE_DRIVER:-devicemapper}

REPEAT_TESTS=${REPEAT_TESTS:-1}
PARALLEL_TESTS=${PARALLEL_TESTS:-1}
FAIL_FAST=${FAIL_FAST:-false}

# choose something relatively unique to avoid intersection with other people runs
# tag would prefix cloud resource groups for your test runs
TAG=${TAG:-$(id -run)}

# what should happen with provisioned VMs on individual test success or failure
DESTROY_ON_SUCCESS=${DESTROY_ON_SUCCESS:-true}
DESTROY_ON_FAILURE=${DESTROY_ON_FAILURE:-true}

# PIN robotest version if needed
ROBOTEST_VERSION=${ROBOTEST_VERSION:-latest}

# which cloud to use : aws or azure
# define rest of the keys in the CLOUD_CONFIG below
DEPLOY_TO=${DEPLOY_TO:-azure}

CLOUD_CONFIG="
installer_url: ${INSTALLER_URL}
script_path: /robotest/terraform/${DEPLOY_TO}
state_dir: /robotest/state
cloud: ${DEPLOY_TO}
aws:
  access_key: ${AWS_ACCESS_KEY}
  secret_key: ${AWS_SECRET_KEY}
  ssh_user: ubuntu
  key_path: /robotest/config/ops.pem
  key_pair: ${AWS_KEYPAIR}
  region: ${AWS_REGION}
  vpc: Create New
  docker_device: /dev/xvdb
azure: 
  subscription_id: ${AZURE_SUBSCRIPTION_ID}
  client_id: ${AZURE_CLIENT_ID}
  client_secret: ${AZURE_CLIENT_SECRET}
  tenant_id: ${AZURE_TENANT_ID}
  vm_type: Standard_F4s
  location: westus
  ssh_user: robotest
  key_path: /robotest/config/ops.pem
  authorized_keys_path: /robotest/config/ops_rsa.pub
  docker_device: /dev/sdd
"
check_args () {
	for v in $@ 
	do
		if [ -z "${!v}" ] ; then 
			echo "variable \$$v must be set"
			ABORT=true
		fi
	done

	if [ ! -z $ABORT ] ; then 
		echo 'Usage: run.sh installer_url tests' ;
		exit 1 ;
	fi

}

check_args INSTALLER_URL RUN_TEST SSH_KEY SSH_PUB 

if [ $DEPLOY_TO == "aws" ]; then
	check_args AWS_REGION AWS_KEYPAIR AWS_SECRET_KEY AWS_ACCESS_KEY
elif [ $DEPLOY_TO == "azure" ]; then 
	check_args AZURE_TENANT_ID AZURE_CLIENT_SECRET AZURE_CLIENT_ID AZURE_SUBSCRIPTION_ID
else
	echo Unsupported deployment cloud $DEPLOY_TO
fi

#
# you generally don't need to change anything beyond this line
# 

P=$(pwd)
export REPORT_FILE=$(date '+%m%d-%H%M')
mkdir -p ${P}/wd_suite/state/${TAG}

set -o xtrace

docker run \
	-v ${P}/wd_suite/state:/robotest/state \
	-v ${SSH_KEY}:/robotest/config/ops.pem \
	-v ${SSH_PUB}:/robotest/config/ops_rsa.pub \
	${ROBOTEST_DEV:+'-v' "${P}/build/robotest-suite:/usr/bin/robotest-suite"} \
	quay.io/gravitational/robotest-suite:${ROBOTEST_VERSION} \
	robotest-suite -test.timeout=48h -test.v \
	-test.parallel=${PARALLEL_TESTS} -repeat=${REPEAT_TESTS} -fail-fast=false \
	-provision="$CLOUD_CONFIG" \
	-resourcegroup-file=/robotest/state/alloc.txt \
	-destroy-on-success=${DESTROY_ON_SUCCESS} -destroy-on-failure=${DESTROY_ON_FAILURE} -always-collect-logs=true \
	-tag=${TAG} -suite=sanity -os=${TEST_OS} -storage-driver=${STORAGE_DRIVER} \
	"${RUN_TEST}" \
	2>&1 | tee ${P}/wd_suite/state/${TAG}/${REPORT_FILE}.txt

echo ${P}/wd_suite/state/${REPORT_FILE}.txt
