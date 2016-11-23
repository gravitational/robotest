package framework

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gravitational/configure"
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/terraform"
	"github.com/gravitational/robotest/infra/vagrant"
	"github.com/gravitational/robotest/lib/loc"
	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
)

// ConfigureFlags registers common command line flags, parses the command line
// and interprets the configuration
func ConfigureFlags() {
	registerCommonFlags()

	flag.Parse()

	initLogger(debugFlag)

	confFile, err := os.Open(configFile)
	if err != nil {
		Failf("failed to read configuration from %q: %v", configFile, err)
	}
	defer confFile.Close()
	err = newFileConfig(confFile)
	if err != nil {
		Failf("failed to read configuration from %q: %v", configFile, err)
	}

	stateFile, err := os.Open(stateConfigFile)
	if err != nil && !os.IsNotExist(err) {
		Failf("failed to read configuration state from %q", stateConfigFile)
	}
	if err == nil {
		defer stateFile.Close()
		err = newFileState(stateFile)
		if err != nil {
			Failf("failed to read configuration state from %q", stateConfigFile)
		}
	}

	if testState != nil {
		TestContext.OpsCenterURL = testState.OpsCenterURL
		TestContext.Provisioner = testState.Provisioner
		TestContext.StateDir = testState.StateDir
	}

	if wizardFlag {
		TestContext.Wizard = wizardFlag
		// Void test state in wizard mode
		testState = nil
	}
	if provisionerName != "" {
		TestContext.Provisioner = provisionerType(provisionerName)
	}
	if teardownFlag {
		TestContext.Teardown = teardownFlag
	}
	if dumpFlag {
		TestContext.DumpCore = dumpFlag
	}

	if TestContext.Teardown || TestContext.DumpCore {
		// Skip tests for this operation
		config.GinkgoConfig.SkipString = ".*"
	}

	log.Debugf("[CONFIG]: %#v", TestContext)
	if testState != nil {
		log.Debugf("[STATE]: %#v", testState)
	}
}

func (r *TestContextType) Validate() error {
	var errors []error
	if TestContext.Wizard && TestContext.Onprem.InstallerURL == "" {
		errors = append(errors, trace.BadParameter("installer URL is required in wizard mode"))
	}
	if TestContext.AWS.IsEmpty() && TestContext.Onprem.IsEmpty() {
		errors = append(errors, trace.BadParameter("either AWS or Onprem is required"))
	}
	if !r.Onprem.IsEmpty() && r.NumInstallNodes > r.Onprem.NumNodes {
		errors = append(errors, trace.BadParameter("cannot install on more nodes than the cluster capacity: %v > %v",
			r.NumInstallNodes, r.Onprem.NumNodes))
	}
	if !r.Onprem.IsEmpty() && r.NumInstallNodes == 0 {
		// TODO: maybe set install nodes = node-1 by default if nodes > 1
		r.NumInstallNodes = r.Onprem.NumNodes
	}
	if r.NumInstallNodes == 0 {
		errors = append(errors, trace.BadParameter("number of install nodes is required"))
	}
	return trace.NewAggregate(errors...)
}

func Failf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Error(msg)
	ginkgo.Fail(nowStamp()+": "+msg, 1)
}

// TestContext defines the global test configuration for the test run
var TestContext = &TestContextType{}

// testState defines an optional state configuration that allows the test runner
// to use state from previous runs
var testState *TestState

// TestContextType defines the configuration context of a single test run
type TestContextType struct {
	// Wizard specifies whether to use wizard to bootstrap cluster
	Wizard bool `json:"wizard" env:"ROBO_WIZARD"`
	// Provisioner defines the type of provisioner to use
	Provisioner provisionerType `json:"provisioner" env:"ROBO_WIZARD"`
	// DumpCore defines a command to collect all installation/operation logs
	DumpCore bool `json:"-"`
	// StateDir specifies the location for test-specific temporary data
	StateDir string `json:"-"`
	// Teardown specifies if the cluster should be destroyed at the end of this
	// test run
	Teardown bool `json:"-"`
	// ReportDir defines location to store the results of the test
	ReportDir string `json:"report_dir" env:"ROBO_REPORT_DIR"`
	// ClusterName defines the name to use for domain name or state directory
	ClusterName string `json:"cluster_name" env:"ROBO_CLUSTER_NAME"`
	// OpsCenterURL defines the URL of the existing Ops Center.
	// This is a requirement for all browser-based tests
	OpsCenterURL string `json:"ops_url" env:"ROBO_CLUSTER_NAME"`
	// Application defines the application package to test
	Application *loc.Locator `json:"application" env:"ROBO_APP"`
	// Login defines the login details to access the Ops Center
	Login Login `json:"login"`
	// ServiceLogin defines the login parameters for service access to the Ops Center
	ServiceLogin ServiceLogin `json:"service_login"`
	// NumInstallNodes defines the subset of nodes to use for installation.
	NumInstallNodes int `json:"install_nodes" env:"ROBO_NUM_INSTALL_NODES"`
	// AWS defines the AWS-specific test configuration
	AWS AWSConfig `json:"aws"`
	// Onprem defines the test configuration for bare metal tests
	Onprem OnpremConfig `json:"onprem"`
}

// Login defines Ops Center authentication parameters
type Login struct {
	Username string `json:"username" env:"ROBO_USERNAME"`
	Password string `json:"password" env:"ROBO_PASSWORD"`
	// AuthProvider specifies the authentication provider to use for login.
	// Available providers are `email` and `gogole`
	AuthProvider string `json:"auth_provider" env:"ROBO_AUTH_PROVIDER"`
}

// ServiceLogin defines authentication options for Ops Center service access
type ServiceLogin struct {
	Username string `json:"username" env:"ROBO_SERVICE_USERNAME"`
	Password string `json:"password" env:"ROBO_SERVICE_PASSWORD"`
}

func (r ServiceLogin) IsEmpty() bool {
	return r.Username == "" && r.Password == ""
}

// AWSConfig describes AWS EC2 test configuration
type AWSConfig struct {
	AccessKey string `json:"access_key" env:"ROBO_AWS_ACCESS_KEY"`
	SecretKey string `json:"secret_key" env:"ROBO_AWS_SECRET_KEY"`
	// Region specifies the EC2 region to install into
	Region string `json:"region" env:"ROBO_AWS_REGION"`
	// KeyPair specifies the name of the SSH key pair to use for provisioning
	// nodes
	KeyPair string `json:"key_pair" env:"ROBO_AWS_KEY_PAIR"`
	// VPC defines the Amazon VPC to install into.
	// Specify "Create new" to create a new VPC for this test run
	VPC string `json:"vpc" env:"ROBO_AWS_VPC"`
	// KeyPath specifies the location of the SSH key to use for remote access.
	// Mandatory only with terraform provisioner
	KeyPath string `json:"key_path" env:"ROBO_AWS_KEY_PATH"`
	// InstanceType defines the type of AWS EC2 instance to boot.
	// Relevant only with terraform provisioner.
	// Defaults are specific to the terraform script used (if any)
	InstanceType string `json:"instance_type" env:"ROBO_AWS_INSTANCE_TYPE"`
}

func (r AWSConfig) IsEmpty() bool {
	return r.AccessKey == "" && r.SecretKey == ""
}

// OnpremConfig defines the test configuration for bare metal tests
type OnpremConfig struct {
	// NumNodes defines the total cluster capacity.
	// This is a total number of nodes to provision
	NumNodes int `json:"nodes" env:"ROBO_NUM_NODES"`
	// InstallerURL defines the location of the installer tarball.
	// Depending on the provisioner - this can be either a URL or local path
	InstallerURL string `json:"installer_url" env:"ROBO_INSTALLER_URL"`
	// ScriptPath defines the path to the provisioner script.
	// TODO: if unspecified, scripts in assets/<provisioner> are used
	ScriptPath string `json:"script_path"  env:"ROBO_SCRIPT_PATH"`
}

func (r OnpremConfig) IsEmpty() bool {
	return r.NumNodes == 0 && r.InstallerURL == "" && r.ScriptPath == ""
}

func registerCommonFlags() {
	// Turn on verbose by default to get spec names
	config.DefaultReporterConfig.Verbose = true
	// Turn on EmitSpecProgress to get spec progress (especially on interrupt)
	config.GinkgoConfig.EmitSpecProgress = true

	flag.StringVar(&configFile, "config-file", "config.json", "Configuration file to use")
	flag.StringVar(&stateConfigFile, "state-file", "config.json.state", "State configuration file to use")
	flag.BoolVar(&debugFlag, "debug", false, "Verbose mode")
	flag.BoolVar(&wizardFlag, "wizard", false, "Run tests in wizard mode")
	flag.BoolVar(&teardownFlag, "destroy", false, "Destroy infrastructure after all tests")
	flag.BoolVar(&dumpFlag, "dumpcore", false, "Collect installation and operation logs into the report directory")
	flag.StringVar(&provisionerName, "provisioner", "", "Provision nodes using this provisioner")
}

func newFileConfig(input io.Reader) error {
	d := json.NewDecoder(input)
	err := d.Decode(&TestContext)
	if err != nil {
		return trace.Wrap(err)
	}

	err = configure.ParseEnv(TestContext)
	if err != nil {
		return trace.Wrap(err)
	}

	err = TestContext.Validate()
	if err != nil {
		return trace.Wrap(err, "failed to validate configuration")
	}

	return nil
}

func newFileState(input io.Reader) error {
	d := json.NewDecoder(input)
	err := d.Decode(&testState)
	if err != nil {
		return trace.Wrap(err)
	}

	err = testState.Validate()
	if err != nil {
		return trace.Wrap(err, "failed to validate state configuration")
	}

	return nil
}

func nowStamp() string {
	return time.Now().Format(time.StampMilli)
}

func initLogger(debug bool) {
	level := log.InfoLevel
	if debug {
		level = log.DebugLevel
	}
	log.StandardLogger().Hooks = make(log.LevelHooks)
	log.SetFormatter(&trace.TextFormatter{TextFormatter: log.TextFormatter{FullTimestamp: true}})
	log.SetOutput(os.Stderr)
	log.SetLevel(level)
}

func provisionerFromConfig(infraConfig infra.Config, stateDir string, provisionerName provisionerType) (provisioner infra.Provisioner, err error) {
	switch provisionerName {
	case provisionerTerraform:
		config := terraform.Config{
			Config:          infraConfig,
			ScriptPath:      TestContext.Onprem.ScriptPath,
			InstallerURL:    TestContext.Onprem.InstallerURL,
			NumNodes:        TestContext.Onprem.NumNodes,
			NumInstallNodes: TestContext.NumInstallNodes,
			AccessKey:       TestContext.AWS.AccessKey,
			SecretKey:       TestContext.AWS.SecretKey,
			KeyPair:         TestContext.AWS.KeyPair,
			SSHKeyPath:      TestContext.AWS.KeyPath,
			InstanceType:    TestContext.AWS.InstanceType,
		}
		err := config.Validate()
		if err != nil {
			return nil, trace.Wrap(err)
		}
		provisioner, err = terraform.New(stateDir, config)
	case provisionerVagrant:
		config := vagrant.Config{
			Config:          infraConfig,
			ScriptPath:      TestContext.Onprem.ScriptPath,
			InstallerURL:    TestContext.Onprem.InstallerURL,
			NumNodes:        TestContext.Onprem.NumNodes,
			NumInstallNodes: TestContext.NumInstallNodes,
		}
		err := config.Validate()
		if err != nil {
			return nil, trace.Wrap(err)
		}
		provisioner, err = vagrant.New(stateDir, config)
	default:
		// no provisioner when the cluster has already been provisioned
		// or automatic provisioning is used
		if provisionerName != "" {
			return nil, trace.BadParameter("unknown provisioner %q", provisionerName)
		}
	}
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return provisioner, nil
}

func provisionerFromState(infraConfig infra.Config, testState TestState) (provisioner infra.Provisioner, err error) {
	err = testState.Validate()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	switch testState.Provisioner {
	case provisionerTerraform:
		config := terraform.Config{
			Config:          infraConfig,
			ScriptPath:      TestContext.Onprem.ScriptPath,
			InstallerURL:    TestContext.Onprem.InstallerURL,
			NumNodes:        len(testState.ProvisionerState.Nodes),
			NumInstallNodes: TestContext.NumInstallNodes,
			AccessKey:       TestContext.AWS.AccessKey,
			SecretKey:       TestContext.AWS.SecretKey,
			KeyPair:         TestContext.AWS.KeyPair,
			SSHKeyPath:      TestContext.AWS.KeyPath,
			InstanceType:    TestContext.AWS.InstanceType,
		}
		err := config.Validate()
		if err != nil {
			return nil, trace.Wrap(err)
		}
		provisioner, err = terraform.NewFromState(config, *testState.ProvisionerState)
	case provisionerVagrant:
		config := vagrant.Config{
			Config:          infraConfig,
			ScriptPath:      TestContext.Onprem.ScriptPath,
			InstallerURL:    TestContext.Onprem.InstallerURL,
			NumNodes:        len(testState.ProvisionerState.Nodes),
			NumInstallNodes: TestContext.NumInstallNodes,
		}
		err := config.Validate()
		if err != nil {
			return nil, trace.Wrap(err)
		}
		provisioner, err = vagrant.NewFromState(config, *testState.ProvisionerState)
	default:
		// no provisioner when the cluster has already been provisioned
		// or automatic provisioning is used
		if testState.Provisioner != "" {
			return nil, trace.BadParameter("unknown provisioner %q", testState.Provisioner)
		}
	}
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return provisioner, nil
}

type provisionerType string

const (
	provisionerTerraform provisionerType = "terraform"
	provisionerVagrant   provisionerType = "vagrant"
)

// configFile defines the configuration file to use for the tests
var configFile string

// stateConfigFile defines the state configuration file to use for the tests
var stateConfigFile string

// debugFlag defines whether to run in verbose mode
var debugFlag bool

// wizardFlag defines whether to run tests in wizard mode
var wizardFlag bool

// provisionerName defines the provisioner to use to provision nodes in the test cluster
var provisionerName string

// teardownFlag defines if the cluster should be destroyed
var teardownFlag bool

// dumpFlag defines whether to collect installation and operation logs
var dumpFlag bool
