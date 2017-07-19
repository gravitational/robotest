#!/bin/bash

set -eu -o pipefail

#
# installer could be local .tar installer or s3:// or http(s) URL
#

if [ -f $INSTALLER_URL ] ; then
	INSTALLER_FILE='/robotest/installer.tar'
fi

# OS could be ubuntu,centos,rhel 
# storage driver: could be devicemapper,loopback,overlay,overlay2 
#  separate multiple values by comma for OS and storage driver
TEST_OS=${TEST_OS:-ubuntu}
STORAGE_DRIVER=${STORAGE_DRIVER:-devicemapper}

REPEAT_TESTS=${REPEAT_TESTS:-1}
PARALLEL_TESTS=${PARALLEL_TESTS:-1}
FAIL_FAST=${FAIL_FAST:-false}
ALWAYS_COLLECT_LOGS=${ALWAYS_COLLECT_LOGS:-true}

# choose something relatively unique to avoid intersection with other people runs
# tag would prefix cloud resource groups for your test runs
TAG=${TAG:-$(id -run)}

# what should happen with provisioned VMs on individual test success or failure
DESTROY_ON_SUCCESS=${DESTROY_ON_SUCCESS:-true}
DESTROY_ON_FAILURE=${DESTROY_ON_FAILURE:-true}

# PIN robotest version if needed
ROBOTEST_VERSION=${ROBOTEST_VERSION:-stable}

check_files () {
	ABORT=
	for v in $@ ; do
		if [ ! -f "${v}" ] ; then 
			echo "${v} does not exist"
			ABORT=true
		fi
	done

	if [ ! -z $ABORT ] ; then 
		exit 1 ;
	fi
}

if [ $DEPLOY_TO != "azure" ] && [ $DEPLOY_TO != "aws" ] ; then
	echo "Unsupported deployment cloud ${DEPLOY_TO}"
	exit 1
fi

if [ $DEPLOY_TO == "aws" ] || [[ $INSTALLER_URL = 's3://'* ]] || [[ ${UPGRADE_FROM:-} = 's3://'* ]]; then
check_files ${SSH_KEY} 
AWS_CONFIG="aws:
  access_key: ${AWS_ACCESS_KEY}
  secret_key: ${AWS_SECRET_KEY}
  ssh_user: ubuntu
  key_path: /robotest/config/ops.pem
  key_pair: ${AWS_KEYPAIR}
  region: ${AWS_REGION}
  vpc: Create New
  docker_device: /dev/xvdb"
fi

if [ $DEPLOY_TO == "azure" ] ; then 
check_files ${SSH_KEY} ${SSH_PUB}
AZURE_CONFIG="azure: 
  subscription_id: ${AZURE_SUBSCRIPTION_ID}
  client_id: ${AZURE_CLIENT_ID}
  client_secret: ${AZURE_CLIENT_SECRET}
  tenant_id: ${AZURE_TENANT_ID}
  vm_type: ${AZURE_VM}
  location: ${AZURE_REGION}
  ssh_user: robotest
  key_path: /robotest/config/ops.pem
  authorized_keys_path: /robotest/config/ops_rsa.pub
  docker_device: /dev/sdd"
fi

if [ -n "${GCL_PROJECT_ID:-}" ] ; then
	check_files ${GOOGLE_APPLICATION_CREDENTIALS}
fi

CLOUD_CONFIG="
installer_url: ${INSTALLER_FILE:-${INSTALLER_URL}}
script_path: /robotest/terraform/${DEPLOY_TO}
state_dir: /robotest/state
cloud: ${DEPLOY_TO}
${AWS_CONFIG:-}
${AZURE_CONFIG:-}
"

# additional context-dependent arguments
args=()

if [ -n "${UPGRADE_FROM:-}" ] ; then
  args+=("-from=${UPGRADE_FROM}")
fi

# will make verbose logging to console, pass -test.v if needed
LOG_CONSOLE=${LOG_CONSOLE:-''}
DOCKER_RUN_FLAGS=${DOCKER_RUN_FLAGS:-''}

P=$(pwd)
export REPORT_FILE=$(date '+%m%d-%H%M')
mkdir -p ${P}/wd_suite/state/${TAG}

set -o xtrace

exec docker run ${DOCKER_RUN_FLAGS} \
	-v ${P}/wd_suite/state:/robotest/state \
	-v ${SSH_KEY}:/robotest/config/ops.pem \
	${AZURE_CONFIG:+'-v' "${SSH_PUB}:/robotest/config/ops_rsa.pub"} \
	${ROBOTEST_DEV:+'-v' "${P}/assets/terraform:/robotest/teraform"} \
	${ROBOTEST_DEV:+'-v' "${P}/build/robotest-suite:/usr/bin/robotest-suite"} \
	${INSTALLER_FILE:+'-v' "${INSTALLER_URL}:${INSTALLER_FILE}"} \
	${GCL_PROJECT_ID:+'-v' "${GOOGLE_APPLICATION_CREDENTIALS}:/robotest/config/gcp.json" '-e' 'GOOGLE_APPLICATION_CREDENTIALS=/robotest/config/gcp.json'} \
	quay.io/gravitational/robotest-suite:${ROBOTEST_VERSION} \
	robotest-suite -test.timeout=48h ${LOG_CONSOLE} \
	${GCL_PROJECT_ID:+"-gcl-project-id=${GCL_PROJECT_ID}"} \
	-test.parallel=${PARALLEL_TESTS} -repeat=${REPEAT_TESTS} -fail-fast=${FAIL_FAST} \
	-provision="${CLOUD_CONFIG}" -always-collect-logs=${ALWAYS_COLLECT_LOGS} \
	-resourcegroup-file=/robotest/state/alloc.txt \
	-destroy-on-success=${DESTROY_ON_SUCCESS} -destroy-on-failure=${DESTROY_ON_FAILURE}  \
	-tag=${TAG} -suite=sanity -os=${TEST_OS} -storage-driver=${STORAGE_DRIVER} \
	"${args[@]}" \
	$@
