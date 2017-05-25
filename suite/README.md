# Gravity CLI test suite

### Configuration File
```
cloud_provider: # aws or azure
installer_url: # could be local file path, http://, https:// or s3://bucket/path URL 
script_path: /robotest/terraform/azure # packaged with docker container, but you can override if needed using -v
docker_device: /dev/sdd # /dev/sdd for azure, /dev/xvdb for aws
aws:
  access_key: # see [AWS EC2 docs](http://docs.aws.amazon.com/general/latest/gr/managing-aws-access-keys.html)
  secret_key: 
  ssh_user: ubuntu # OS distribution may override 
  key_path: /robotest/config/ssh_private_key.pem # we use it to SSH to hosts
  key_pair: # public part of SSH key is stored on AWS EC2, see its docs
  region: us-west-1 # if you change it, you need also update AMI in terraform/aws/os.tf
  vpc: Create new # leave it like that
  docker_device: /dev/xvdb 
azure: 
  subscription_id: # see [Azure docs](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal) for next for params
  client_id: 
  client_secret: 
  tenant_id: 
  vm_type: Standard_DS2_v2 
  location: westus
  ssh_user: robotest
  key_path: /robotest/config/ssh_private_key.pem
  authorized_keys_path: /robotest/config/ssh_public
  docker_device: /dev/sdd

```

### Shell command
```
docker run \
	-v $(pwd)/robotest_state:/robotest/state \
    -v $(pwd)/app-installer.tar:/robotest/app-installer.tar \
	-v ~/.ssh/id_rsa.pub:/robotest/config/ssh_public \
    -v $(pwd)/config.yaml:/robotest/config/config.yaml \
    -v ~/.ssh/id_rsa:/robotest/config/ssh_private_key.pem \
	quay.io/gravitational/robotest-suite:latest \
	-config /robotest/config/config.yaml \
	-dir=/robotest/state -resourcegroupfile=/robotest/state/alloc.txt \
	-destroy-on-success=true -destroy-on-failure=true \
    -tag=RTJ-${BRANCH//\//-} -suite=sanity -set=${TEST_SETS}
```

### Runtime Args

Every test creates its own resource group on cloud provider, and removes it after test is complete. 

1. `config` : see sample YAML config file above.
2. `dir` : directory to keep state, its recommended to map it from host.
3. `resourcegroupfile` will keep list of resource groups created on cloud.
4. `destroy-on-success` whether to remove VMs after successful test run.
5. `destroy-on-failure` whether to remove VMs after test failure. 
6. `tag` all resource groups will start from this. Choose something unique.
7. `suite` : `sanity`, `stress`, `regression`. Use `sanity`. 
8. `set` whether to run a [specific test set](https://github.com/gravitational/robotest/blob/master/suite/sanity/sanity.go) or all. 