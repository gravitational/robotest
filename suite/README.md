## Gravity CLI test suite

A recommended way to launch CLI suite is by using either `latest` or `stable` version of docker container 
and defining few dynamic configuration variables plus necessary cloud environment parameters. 

```bash
#!/bin/bash

# Robotest dynamically generates test names and corresponding cloud resource groups 
# TAG is used to prefix them. Keep it short (i.e. 4 chars), as cloud resource groups have length limits
export TAG=

# Amount of parallel tests to run. Use it to constraint cloud resource usage to avoid hitting quota.
export PARALLEL_TESTS=1

# How many times each test should be repeated. 
export REPEAT_TESTS=1

# When true, aborts all tests on first failure
export FAIL_FAST=false 

# OS could be ubuntu,centos,redhat 
TEST_OS=${TEST_OS:-ubuntu}

# storage driver: could be devicemapper,loopback,overlay,overlay2 
# separate multiple values by comma for OS and storage driver
STORAGE_DRIVER=${STORAGE_DRIVER:-devicemapper}

# Keep or destroy allocated VMs for each successful or failed test
export DESTROY_ON_SUCCESS=true
export DESTROY_ON_FAILURE=true

# Valid combinations are latest, stable or specific version 
export ROBOTEST_VERSION="stable"
export REPO=quay.io/gravitational/robotest-suite:${ROBOTEST_VERSION}

# Which cloud to deploy. Valid values are aws and azure
export DEPLOY_TO=aws

# Path to SSH key 
export SSH_KEY=

# Path to public part of SSH key, only required for Azure
export SSH_PUB=

# Define to enable all log forwarding to google cloud logger and dashboard
export GCL_PROJECT_ID=kubeadm-167321

# Installer could be a local file path (don't prefix with file://) , s3:// or http(s):// URL
export INSTALLER_URL='s3://s3.gravitational.io/builds/c1b6794-telekube-3.56.4-installer.tar'

set -o pipefail

docker pull ${REPO}
SCRIPT=$(mktemp -d)/runsuite.sh
docker run ${REPO} cat /usr/bin/run_suite.sh > ${SCRIPT}
chmod +x ${SCRIPT}
${SCRIPT} install='{"nodes":1,"flavor":"one"}'

```

## In development

The following commands in gravity repository will build telekube
```
$ make production telekube
$ make run-robotest-suite ROBOTEST_SUITE=scripts/robotest/superlite.txt
```

see various suites defined in `scripts/robotest` folder of `gravity` repository.

## Supported Tests
Every test is passed as argument to launch script as `testname={json}`. Mind the double-quotes for field names.

### Install a cluster

`install`

* `nodes` (uint) number of nodes.
* `flavor` (string) flavor corresponding to number of nodes.
* `remote_support` (bool, default=false) enable remote support via `gravity complete` after install using OPS center and token burned into installer.
* `uninstall` (bool, default=false) uninstall at the end
* `service_uid` (uint, default=gravity default) the uid that planet will run under, see https://gravitational.com/gravity/docs/ver/7.x/pack/#service-user
* `service_gid` (uint, default=gravity default) the gid that planet will run under

`provision` takes same args but will not run any installer, just provision VMs. 

### Install cluster, then expand

`resize`

Inherits parameters from `install`, plus:

* `to` (uint) number of nodes to expand (or shrink) to
* `graceful` (bool, default=false) whether to perform graceful or forced node shrink

### Install cluster, expand by one, shrink by one

`shrink`

Inherits all parameters from `install`.

### Install cluster, then upgrade

`upgrade` inherits parameters from `install`.

* `upgrade_from` initial installer to use

### Recover cluster nodes

`recover` inherits `install` parameters.

* `roles` (array) `["apimaster","clmaster","clbackup","worker"]` will sequentially locate and replace nodes with given role
* `recycle` (bool) if true, a clean node will be used for each operation replacement, if false then +1 node would be created in addition to `nodes` parameters and will sequentially be replaced as per nodes. Note the `worker` is no-op for cluster with <= 3 nodes.
* `expand_before_shrink` (bool) expand cluster before node removal or after.
* `pwroff_before_remove` (bool) if true, then node would be `poweroff -f` before node replacement. Cannot be combined with `recycle=true`

`recoverV` will generate a combination of `recover` parameterized tests.

### Post installer transfer script
When a certain application may require extra setup after provisioning and installer transfer is complete, this could be achieved by passing extra parameters to tests: 
```json
"script" : {
    "url" : "local path, s3 or http(s) url",
    "args" : ["args", "to", "script"],
}
```

## Cloud Environment Configuration

Currently deployment to AWS and Azure is supported. 

### AWS Configuration

When deploying to AWS or using S3:// installer URLs, you need define `AWS_REGION, AWS_KEYPAIR, AWS_ACCESS_KEY, AWS_SECRET_KEY` environment variables. See [AWS EC2 docs](http://docs.aws.amazon.com/general/latest/gr/managing-aws-access-keys.html) for details.

In order to use AWS VPC networking, instances will be assigned IAM Instance Profile `robotest-node` with the following policy. Note this IAM Instance Profile is not created dynamically.

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:Describe*",
                "ec2:AttachVolume",
                "ec2:DetachVolume"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "ec2:CreateRoute",
                "ec2:DeleteRoute",
                "ec2:ReplaceRoute"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeRouteTables",
                "ec2:DescribeInstances"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "elasticloadbalancing:DescribeLoadBalancers"
            ],
            "Resource": "*"
        }
    ]
}
```

### Azure Configuration
When deploying to Azure, you need define `AZURE_SUBSCRIPTION_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_TENANT_ID` authentication variables. See [Azure docs](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal) for more details. 

* `AZURE_REGION` are comma-separated regions to deploy to; Use `az account list-locations` for options.
* `AZURE_VM` is [VM size](https://docs.microsoft.com/en-us/azure/virtual-machines/linux/sizes); default is `Standard_F4s`. Use `az vm list-sizes --location ${AZURE_REGION}` to check which VMs are available.

### Cloud Logging
Robotest can optionally send detailed execution logs to Google Cloud Logging platform.

1. [Create and generate key for the service account](https://cloud.google.com/docs/authentication/getting-started) and set `GOOGLE_APPLICATION_CREDENTIALS` environment variable accordingly.
2. Assign `Logging/Log Writer` and `Pub-Sub/Topic Writer` permissions to the service account.
3. Enable [Cloud Logging](https://console.cloud.google.com/logs/viewer) project and set `GCL_PROJECT_ID` env variable to [google project ID](https://console.cloud.google.com/iam-admin/settings/project).

### Using local files
Robotest is executed from within a container, and therefore cannot access any local files directly. When you need to pass local file as installer tarball, mount them individually or a holding directory using `EXTRA_VOLUME_MOUNTS` variable, following docker's [volume mount](https://docs.docker.com/engine/admin/volumes/bind-mounts/) semantics `-v local_path:container_path`.

### Cleaning up leaked resources
If robotest fails to clean up all cloud resources (e.g. due to interruption during a run), the terraform files in the state directory can be used to destroy these resources.
For example:

```
$ cd wd_suite/state/7bdc964/install-1/redhat7/overlay2/1n/tf/
$ ls
bootstrap  config.tf  network.tf  node.tf  os.tf  output.tf  robotest.tfvars.json  terraform.log  terraform.tfstate  terraform.tfstate.backup  versions.tf
$ terraform destroy -var-file robotest.tfvars.json
data.template_file.bootstrap: Refreshing state...
data.google_compute_subnetwork.robotest: Refreshing state...
data.google_compute_network.robotest: Refreshing state...
data.google_compute_zones.available: Refreshing state...
random_shuffle.zones: Refreshing state... [id=-]
google_compute_disk.etcd[0]: Refreshing state... [id=projects/robotest/zones/us-west1-b/disks/robotest-12345678-disk-etcd-0]
google_compute_disk.boot[0]: Refreshing state... [id=projects/robotest/zones/us-west1-b/disks/robotest-12345678-disk-boot-0]
google_compute_instance.node[0]: Refreshing state... [id=projects/robotest/zones/us-west1-b/instances/robotest-12345678-node-0]
google_compute_instance_group.robotest: Refreshing state... [id=projects/robotest/zones/us-west1-b/instanceGroups/robotest-12345678-node-group]

An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
  - destroy

// snip

Plan: 0 to add, 0 to change, 5 to destroy.

Do you really want to destroy all resources?
  Terraform will destroy all your managed infrastructure, as shown above.
  There is no undo. Only 'yes' will be accepted to confirm.

  Enter a value:
```
