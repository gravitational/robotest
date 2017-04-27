# robotest

`robotest` is a set of integration tests for the gravity platform.
It is implemented as a testing package that is built as a binary with custom command-line to drive test execution:

```shell
$ ./robotest -h
Usage of ./robotest:
  ...
  -config string
    	Configuration file to use (default "config.yaml")
  -debug
    	Verbose mode
  -destroy
    	Destroy infrastructure after all tests
  -output
        Display current state only
  -provisioner string
    	Provision nodes using this provisioner
  -report
    	Collect installation and operation logs into the report directory
  -mode wizard
    	Run robotest in specific mode. Supported modes: [wizard,`provision`]
  ...
  -ginkgo.focus string
    	If set, ginkgo will only run specs that match this regular expression.
  -ginkgo.skip string
    	If set, ginkgo will only run specs that do not match this regular expression.
  -ginkgo.v
    	If set, default reporter print out all specs as they begin.
```

The tool supports stateful mode of operation when there're bootstrapping (i.e. creating infrastructure) and cleanup phases (i.e. destroying the infrastructure) and as such can fit many more scenarios than a monolithic design would allow.

## Building

To build, invoke from the repository directory:

```
go test -i -c ./e2e
```

## Configuration

Configuration is stored in a YAML file that can be specified with `-config` on the command line. It defaults to `config.yaml`.
Here's an example configuration:

```yaml
report_dir: /tmp/robotest-reports
state_dir: /tmp/state-dir
cluster_name: test
ops_url: https://localhost:33009
application: gravitational.io/k8s-aws:0.0.0+latest
license: "application license"
web_driver_url: http://localhost:9515  # chromedriver instance
flavor_label: "2 nodes"
provisioner: vagrant
login:
    username: user
    password: password
    auth_provider: google
service_login:
    username: robotest@example.com
    password: robotest!
extensions:
    install_timeout: 1h
    backup_config:
        addr: 192.168.0.2
        path: /var/lib/backup/backup.tar.gz
aws:
    access_key: "access key"
    secret key: "secret key"
    region: us-east-1
    key_pair: test
    ssh_user: centos
    vpc: "Create new"
    key_path: /path/to/SSH/key
onprem:
    script_path: /home/robotest/assets/vagrant/Vagrantfile
    installer_url: /home/robotest/assets/installer/installer.tar.gz
    nodes: 2
    docker_device: /dev/sdb
```

 * `report_dir` specifies an optional location of the log files which are always collected during teardown or, manually, with `-report` command
 * `state_dir` specifies the location for test-specific data. For example, terraform state files.
 * `provisioner` specifies the type of provisioner to use
 * `cluster_name` specifies the name of the cluster (and domain) to create for tests
 * `ops_url` specifies the URL of an active Ops Center to run tests against (see note below on [Wizard mode](#wizard-mode))
 * `application` specifies the name of the application package to run tests with (see note below on [Wizard mode](#wizard-mode))
 * `web_driver_url` specifies an optional URL of the web driver to use, e.g. http://localhost:4444/wd/hub for selenium
  or http://localhost:9515 for chrome driver
 * `login` block specifies user details for authenticating to Ops Center (see note below on [Wizard mode](#wizard-mode))
 * `service_login` specifies details of a service user to use to programmatically access Ops Center from the command line. This can be a
  user specifically created for tests. The user will be used to connect to the Ops Center and query logs or export/import application packages
  as required by tests.
 * `aws` specifies a [block](#aws-configuration) of parameters for AWS-based test scenarios.
 * `onprem` specifies a [block](#onprem-configuration) of parameters for bare metal tests.
 * `extensions` specifies a [block](#step-configuration) of parameters for arbitrary test steps.


### Ops Center login

This section specifies parameters to login into Ops Center using a browser.
The `auth_provider` is one of [`google`, `email`].


### AWS configuration

All parameters in this section should be self-explanatory.
The `vpc` parameter specifies whether to use an existing or create a new VPC.
The valid values are names of VPCs or a special value `Create new` to indicate the fact that a new VPC is to be created.

### Onprem configuration

The bare metal provisioners can work in two modes - creating an infrastructure and optionally executing an installer to
run tests from an installer tarball. See below on details about the [Wizard mode](#wizard-mode).

The `installer_url` specifies either the URL of the installer tarball to download (as required by the `terraform` provisioner) or
a path to a local tarball for `vagrant`. The `installer_url` is optional and is only required for [Wizard mode](#wizard-mode).

The `docker_device` specifies the device for docker with devicemapper driver. With empty value for this param docker will use loopback device, which is not recommended for prodaction usage.

The `nodes` parameter specifies the total cluster capacity (e.g. the number of total nodes to provision).
Note the `flavor_label` paramater in the global configuration section - this parameter specifies the actual
installation flavor which determines the number of nodes used for installation. The selected flavor should not exceed the number of `nodes`.

### Step configuration

Extensions block allows configuration of individual test steps.
Currently it supports the following attributes:

  * `install_timeout` specifies the time allotted for the install operation to complete. It defaults to some value if unspecified.
  * `backup_config` specifies configuration for backup/restore tests. `backup_config` supports two attributes:
    * `addr` specifies address of node, where backup/restore test will be executed.
    * `path` specifies path on node with `addr`, where backup file is stored. For restore test - robotest will read file on that path.

## Creating infrastructure (bare metal tests)

The tool support two provisioners out of the box: [terraform] and [vagrant].
The bundled scripts can provision cluster of arbitrary size but the size configuration is static and must be configured before hand.
To choose a provisioner, simply run the tool with `-provisioner <name>` and configure the path to the script file to use.
There're several provisioner scripts available in this repository - for both types, `terraform` and `vagrant`:

```
assets/
├── terraform
│     ├── terraform.tf
│     └── terraform_noinstaller.tf
└── vagrant
      └── Vagrantfile
```

`terraform_noinstaller.tf` is a variation w/o downloading and unpacking an installer tarball.


### Creating infrastructure (terraform)

To provision a terraform-based infrastructure, configure the `onprem` section of the configuration file and invoke the binary:

```yaml
onprem:
    script_path: /home/robotest/assets/terraform/terraform.tf
    installer_url: s3://infra.example.com/installers/v0.0.1/installer.tar.gz
    nodes: 1
    docker_device: /dev/xvdb
```

```shell
$ ./robotest -provisioner=terraform -config=config.yaml -ginkgo.focus=`Onprem Install`
```

### Creating infrastructure (vagrant)

To provision a vagrant-based infrastructure, configure the `onprem` section of the configuration file and invoke the binary:

```yaml
onprem:
    script_path: /home/robotest/assets/vagrant/Vagrantfile
    installer_url: /home/robotest/assets/installer/installer.tar.gz
    nodes: 2
    docker_device: /dev/sdb
```

```shell
$ ./robotest -provisioner=vagrant -config=config.yaml -ginkgo.focus=`Onprem Install`
```


## Provision mode

To only provision infrastructure invoke the tool with additional `-mode=provision` flag.

```shell
$ ./robotest -provisioner=vagrant -config=config.yaml -mode=provision -ginkgo.skip=`.*`
```

## Wizard mode

If the tests are to be run against an installer tarball of a particular application, then the tool is invoked with an additional
`-mode=wizard` flag:


```shell
$ ./robotest -provisioner=vagrant -config=config.yaml -mode=wizard -ginkgo.focus=`Onprem Install`
```

This changes the operation mode to provision a cluster, choose a node for installer and start the installer - all done automatically before
any tests are run.

In this mode, the installer creates and manages an ephemeral Ops Center instance - it is not mandatory to specify Ops Center URL in configuration.
Just as with the Ops Center URL, the application to install is automatically discovered from the installer - `application` configuration option does
not apply to wizard mode and is ignored.

The `login` block for configuration is not necessary for the wizard mode as the installer has a built-in user that will automatically be used for login.


## Integration Tests

The package uses [ginkgo] as a test runner. The tests are split into [specs] (independent pieces that can be tested individually and in arbitrary order).
We differentiate the tests in two directions: AWS and bare metal.

Here're the relevant top-level test specs:

  * `Onprem Installation` specifies the context for installing an application on bare metal (including AWS bare metal - e.g. as provisioned by `terraform`)
  * `AWS Installation` specifies the context for installing an application on AWS cloud (using automatic provisioning)

These two test specs should be used to bootstrap a test - i.e. create an infrastructure and install an application.
So every test run should start with either:

```
$ ./robotest -ginkgo.focus='Onprem Install' ...
```
or

```
$ ./robotest -ginkgo.focus='AWS Install' ...
```

to setup the cluster for further tests.

`ginkgo.focus` specifies a regular expression to use as an anchor to search for specs to execute. Its counterpart is `ginkgo.skip` which specifies
the specs to skip. Without this option, the default behavior is to execute **all** available test specs in **arbitrary** order (although by default [ginkgo] permutes only the top-level contexts).

### Running in-between tests

After the
structure has been prepared and the application installed, one can run a further set of tests that require an infrastructure and
an application:

```
$ ./robotest -gingko.focus='Site Update'
$ ./robotest -gingko.focus='Site Servers'
$ ./robotest -gingko.focus='Application Backup Test'
$ ./robotest -gingko.focus='Application Restore Test'
```

### Test cleanup

After executing all tests, the infrastructure can be destroyed by invoking the test binary with `-destroy`:

```
$ ./robotest -destroy
```

This is only relevant for bare metal configurations. The automatically provisioned AWS clusters can only cleaned up by running the `uninstall` test.


## Usage of Docker image

Docker image includes `robotest`, `terraform` and `chromedriver` binaries. Robotest inside docker image can work only with `terraform` provider
and only with config in YAML or JSON format.
How to use it:

``` shell
docker run -v /path/to/working/directory/:/robotest/ \
    -v /path/to/gravity/binary:/usr/bin/gravity \
    -v /tmp/robotest-report/:/tmp/robotest-report/ \
     quay.io/gravitational/robotest-standalone:1.0.0 -config /robotest/config/config.yaml + other robotest tool flags
```

Working directory on host(`/path/to/working/directory`) will contain test state and should have specific structure:

```
workdir/
├── config
│   ├── config.yaml or config.json
│   ├── terraform.tf
│   └── ssh.key
└── state
```

Example of `config.yaml` for that file structure:

``` yaml
report_dir: /tmp/robotest-reports
state_dir: /robotest/state
cluster_name: test
ops_url: https://localhost:33009
application: gravitational.io/k8s-aws:0.0.0+latest
license: "application license"
web_driver_url: http://localhost:9515  # local chromedriver instance
flavor_label: "2 nodes"
provisioner: terraform
login:
    username: user
    password: password
    auth_provider: google
service_login:
    username: robotest@example.com
    password: robotest!
extensions:
    install_timeout: 1h
aws:
    access_key: "access key"
    secret key: "secret key"
    region: us-east-1
    key_pair: test
    ssh_user: centos
    vpc: "Create new"
    key_path: /robotest/config/ssh.key
onprem:
    script_path: /robotest/config/terraform.tf
    installer_url: s3://infra.example.com/installers/v0.0.1/installer.tar.gz
    nodes: 2
    docker_device: /dev/xvdb
```

## Browser-based testing

Currently set of test specs are all browser-based and require a [WebDriver]-compatible implementation ([selenium] or [chrome-driver] are two examples).
If no web driver has been configured, [chrome-driver] will be used.


[//]: # (Footnotes and references)

[WebDriver]: https://w3c.github.io/webdriver/webdriver-spec.html
[selenium]: http://www.seleniumhq.org/
[chromedriver]: https://sites.google.com/a/chromium.org/chromedriver/
[terraform]: https://www.terraform.io/
[vagrant]: https://www.vagrantup.com/
[ginkgo]: https://onsi.github.io/ginkgo/
[specs]: https://onsi.github.io/ginkgo/#structuring-your-specs
