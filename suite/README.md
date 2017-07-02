## Gravity CLI test suite

A recommended way to launch CLI suite is by using either `latest` or `stable` version of docker container 
and defining few dynamic configuration variables plus necessary cloud environment parameters. 

```bash
#!/bin/bash

# Robotest dynamically generates test names and corresponding cloud resource groups 
# TAG is used to prefix them. Keep it short (i.e. 4 chars), as cloud resource groups have length limits
export TAG=sanity/install

# Amount of parallel tests to run. Use it to constraint cloud resource usage to avoid hitting quota.
export PARALLEL_TESTS=1

# How many times each test should be repeated. 
export REPEAT_TESTS=1

# When true, aborts all tests on first failure
export FAIL_FAST=false 

# Keep or destroy allocated VMs for each successful or failed test
export DESTROY_ON_SUCCESS=false
export DESTROY_ON_FAILURE=false

# Valid combinations are latest, stable or specific version 
export ROBOTEST_VERSION="stable"

# Which cloud to deploy. Valid values are aws and azure
export DEPLOY_TO=aws

# Define to enable all log forwarding to google cloud logger and dashboard
export GCL_PROJECT_ID=kubeadm-167321

# Installer could be a local file path (don't prefix with file://) , s3:// or http(s):// URL
export INSTALLER_URL='s3://s3.gravitational.io/builds/c1b6794-telekube-3.56.4-installer.tar'

set -o pipefail

docker run --pull \
  quay.io/gravitational/robotest-suite:${ROBOTEST_VERSION} \
  cat /usr/bin/run_suite.sh | /bin/bash -s 'install={"nodes":1,"flavor":"one"}'

```

## Jenkins

* [Compile and test specific telekube branch](https://jenkins.gravitational.io/view/robotest/job/robotest-run/)
* [Compile and publish Robotest](https://jenkins.gravitational.io/view/robotest/job/Robotest-publish/)

## Supported Tests
Every test is passed as argument to launch script as `testname={json}`. Mind the double-quotes for field names.

### Install a cluster

`install`

* `nodes` (uint) number of nodes.
* `flavor` (string) flavor corresponding to number of nodes.
* `remote_support` (bool, default=false) enable remote support via `gravity complete` after install using OPS center and token burned into installer.
* `uninstall` (bool, default=false) uninstall at the end

`provision` takes same args but will not run any installer, just provision VMs. 

### Install cluster, then resize

`resize` 

Inherits parameters from `install`, plus:

* `to` (uint) number of nodes to expand (or shrink) to
* `graceful` (bool, default=false) whether to perform graceful or forced node shrink

### Install cluster, then upgrade

`upgrade3lts` - current upgrade procedure for 3.x LTS branch. Inherits parameters from `install`. 

* `upgrade_from` initial installer to use

### Replace cluster nodes

`replace` inherits `install` parameters. 

* `roles` (array) `["apimaster","clmaster","clbackup","worker"]` will sequentially locate and replace nodes with given role
* `recycle` (bool) if true, a clean node will be used for each operation replacement, if false then +1 node would be created in addition to `nodes` parameters and will sequentially be replaced as per nodes. Note the `worker` is no-op for cluster with <= 3 nodes.
* `expand_before_shrink` (bool) expand cluster before node removal or after. 
* `pwroff_before_remove` (bool) if true, then node would be `poweroff -f` before node replacement. Cannot be combined with `recycle=true`

`replace_variety` will generate a combination of `replace` parameterized tests.

## Cloud Environment Configuration

Currently deployment to AWS and Azure is supported. 

### AWS Configuration

When deploying to AWS or using S3:// installer URLs, you need define `AWS_REGION, AWS_KEYPAIR, AWS_ACCESS_KEY, AWS_SECRET_KEY` environment variables. See [AWS EC2 docs](http://docs.aws.amazon.com/general/latest/gr/managing-aws-access-keys.html) for details.

### Azure Configuration
When deploying to Azure, you need define `AZURE_SUBSCRIPTION_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_TENANT_ID` variables. See [Azure docs](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal) for more details. 

### Cloud Logging
Robotest can optionally send detailed execution logs to Google Cloud Logging platform.

1. [Create and generate key for the service account](https://cloud.google.com/docs/authentication/getting-started) and set `GOOGLE_APPLICATION_CREDENTIALS` environment variable accordingly.
2. Assign `Logging/Log Writer` and `Pub-Sub/Topic Writer` permissions to the service account.
3. Enable [Cloud Logging](https://console.cloud.google.com/logs/viewer) project and set `GCL_PROJECT_ID` env variable to [google project ID](https://console.cloud.google.com/iam-admin/settings/project).
