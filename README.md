# robotest

`robotest` is a set of integration tests for the gravity platform.
It is implemented as a testing package that is built as a binary with custom command-line to drive test execution:

```shell
$ ./e2e.test -h
Usage of ./e2e.test:
  ...
  -config-file string
    	Configuration file to use (default "config.json")
  -debug
    	Verbose mode
  -destroy
    	Destroy infrastructure after all tests
  -dumpcore
    	Collect installation and operation logs into the report directory
  -provisioner string
    	Provision nodes using this provisioner
  -wizard
    	Run tests in wizard mode
  ...
  -ginkgo.focus string
    	If set, ginkgo will only run specs that match this regular expression.
  -ginkgo.skip string
    	If set, ginkgo will only run specs that do not match this regular expression.
  -ginkgo.v
    	If set, default reporter print out all specs as they begin.
```

The tool supports stateful mode of operation when there're bootstrapping (i.e. creating infrastructure) and cleanup phases (i.e. destroying the infrastructure) and as such can fit many more scenarios than would a monolithic design allow.

## Configuration

Configuration is stored in a JSON file that can be specified with `-config-file` on the command line. It defaults to `config.json`.
Here's an example configuration:

```json
{
    "report_dir": "/tmp/robotest-reports",
    "cluster_name": "test",
    "ops_url": "https://localhost:33009",
    "application": "gravitational.io/k8s-aws:0.0.0+latest",
    "install_nodes": 2,
    "login": {
        "username": "user",
        "password": "password",
        "auth_provider": "google"
    },
    "service_login": {
        "username": "robotest@example.com",
        "password": "robotest!"
    },
    "aws": {
        "access_key": "access key",
        "secret key": "secret key",
        "region": "us-east-1",
        "key_pair": "test",
        "vpc": "Create new"
        "key_path": "/path/to/SSH/key"
    },
    "onprem": {
        "script_path": "/home/robotest/assets/vagrant/Vagrantfile",
        "installer_url": "/home/robotest/assets/installer/installer.tar.gz",
        "nodes": 2
    }
}
```

 * `report_dir` specifies the location of the log files which are always collected during teardown or, manually, with `-dumpcore` command
 * `cluster_name` specifies the name of the cluster (and domain) to create for tests
 * `ops_url` specifies the URL of an active Ops Center to run tests against (see note below on [Wizard mode](#Wizard-mode))
 * `application` specifies the name of the application package to run tests with
 * `login` block specifies user details for authenticating to Ops Center
 * `service_login` specifies details of a service user to use to programmatically access Ops Center from the command line. This can be a
  user specifically created for tests. The user will be used to connect to the Ops Center and query logs or export/import application packages
  as required by tests.
 * `aws` specifies a block of parameters for AWS-based test scenarios.
 * `onprem` specifies a block of parameters for bare metal tests.


### Ops Center login

This section specifies parameters to login into Ops Center using a browser.
The `auth_provider` is one of [`google`, `email`].


### AWS configuration

All parameters in this section should be self-explanatory.
The `vpc` parameter specifies whether to use an existing or create a new VPC.
The valid values are names of VPCs or a special value `Create new` to indicate the fact that a new VPC is to be created.

### Onprem configuration

The bare metal provisioners can work in two modes - creating an infrastructure and optionally executing an installer to
run tests from an installer tarball. See below on details about the wizard mode.

The `installer_url` specifies either the URL of the installer tarball to download (as required by the `terraform` provisioner) or
a path to a local tarball for `vagrant`. The `instalelr_url` is optional and is only required for [Wizard mode](#Wizard-mode).

The `nodes` parameter specifies the total cluster capacity (e.g. the number of total nodes to provision).
Note the `install_nodes` paramater in the global configuration section - this optional parameter specifies how many nodes to
use for the initial installation and must be <= `nodes`. If not specified, it defaults to `nodes`.

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

`terraform_noinstaller.tf` is a variation w/o the installer bootstrap script.


### Creating infrastructure (terraform)

To provision a terraform-based infrastructure, configure the `onprem` section of the configuration file and invoke the binary:

```shell
$ ./e2e.test -provisioner terraform -config-file=config.json -ginkgo.focus=`Onprem Install`
```

### Creating infrastructure (vagrant)

To provision a vagrant-based infrastructure, configure the `onprem` section of the configuration file and invoke the binary:

```shell
$ ./e2e.test -provisioner vagrant -config-file=config.json -ginkgo.focus=`Onprem Install`
```


## Wizard mode

If the tests are to be run against an installer tarball of a particular application, then the tool is invoked with an additional
`wizard` flag:


```shell
$ ./e2e.test -provisioner vagrant -config-file=config.json -wizard -ginkgo.focus=`Onprem Install`
```

This changes the operation mode to provision a cluster, choose a node for installer and start the installer - all done automatically before
any tests are run.


## Integration Tests

The package uses [ginkgo] as a test runner. The tests are split into [specs] (independent pieces that can be tests individually and in arbitrary order).
We differentiate the tests in two directions: AWS and bare metal.

Here're the relevant top-level test specs:

  * `Onprem Installation` specifies the context for installing an application on bare metal (including AWS bare metal - e.g. as provisioned by `terraform`)
  * `AWS Installation` specifies the context for installing an application on AWS cloud (using automatic provisioning)

These two test specs should be used to bootstrap a test - i.e. create an infrastructure and install an application.
So every test run should start with either:

```
$ ./e2e.test -ginkgo.focus='Onprem Install' ...
```
or

```
$ ./e2e.test -ginkgo.focus='AWS Install' ...
```

to setup the cluster for further tests.

`ginkgo.focus` specifies a regular expression to use as an anchor to search for specs to execute. Its counterpart is `ginkgo.skip` which specifies
the specs to skip. Without this option, the default behavior is to execuet _all_ available test specs in _arbitrary_ order (although by default [ginkgo] permutes only the top-level contexts).

### Running in-between tests

After the infrstructure has been prepared and the application installed, one can run a further set of tests that require an infrstructure and
the application:

```
$ ./e2e.test -gingko.focus='Site Update'
$ ./e2e.test -gingko.focus='Site Servers'
```

### Test cleanup

After executing all tests, the infrastructure can be destroyed by invoking the test binary with `-destroy`:

```
$ ./e2e.test -destroy
```

This is only relevant for bare metal configurations. The automatically provisioned AWS clusters can only cleaned up by running the `uninstall` test.


## Browser-based testing

Currently set of test specs are all browser-based and require a [WebDriver]-compatible implementation ([selenium] or [chrome-driver] are two examples).
The choice of the driver is hard-coded and defaults to [chrome-driver].



[//]: # (Footnotes and references)

[WebDriver]: https://w3c.github.io/webdriver/webdriver-spec.html
[selenium]: http://www.seleniumhq.org/
[chrome-driver]: https://sites.google.com/a/chromium.org/chromedriver/
[terraform]: https://www.terraform.io/
[vagrant]: https://www.vagrantup.com/
[ginkgo]: https://onsi.github.io/ginkgo/
[specs]: https://onsi.github.io/ginkgo/#structuring-your-specs
