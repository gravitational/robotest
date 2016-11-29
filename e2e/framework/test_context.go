package framework

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/gravitational/configure"
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/terraform"
	"github.com/gravitational/robotest/infra/vagrant"
	"github.com/gravitational/robotest/lib/loc"
	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
	"github.com/go-yaml/yaml"
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
		Failf("failed to read configuration from %q: %v", configFile, trace.DebugReport(err))
	}

	stateFile, err := os.Open(stateConfigFile)
	if err != nil && !os.IsNotExist(err) {
		Failf("failed to read configuration state from %q", stateConfigFile)
	}
	if err == nil {
		defer stateFile.Close()
		err = newFileState(stateFile)
		if err != nil {
			testState = nil
			Failf("failed to read configuration state from %q", stateConfigFile)
		}
	}

	if testState != nil {
		TestContext.Provisioner = testState.Provisioner
		TestContext.StateDir = testState.StateDir
	}

	if mode == wizardMode {
		TestContext.Wizard = true
		// Void test state in wizard mode
		testState = nil
	} else {
		TestContext.Onprem.InstallerURL = ""
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
		log.Debugf("[STATE]: %#v", testState, testState.ProvisionerState)
		if testState.ProvisionerState != nil {
			log.Debugf("[PROVISIONER STATE]: %#v", *testState.ProvisionerState)
		}
	}
}

func (r *TestContextType) Validate() error {
	var errors []error
	if TestContext.Wizard && TestContext.Onprem.InstallerURL == "" {
		errors = append(errors, trace.BadParameter("installer URL is required in wizard mode"))
	}
	var command bool = teardownFlag || dumpFlag || mode == wizardMode
	if !command && TestContext.Login.IsEmpty() {
		errors = append(errors, trace.BadParameter("Ops Center login is required"))
	}
	if TestContext.ServiceLogin.IsEmpty() {
		log.Warningf("service login not configured - reports will not be collected")
	}
	if TestContext.AWS.IsEmpty() && TestContext.Onprem.IsEmpty() {
		errors = append(errors, trace.BadParameter("either AWS or Onprem is required"))
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
	Wizard bool `json:"wizard" yaml:"wizard" env:"ROBO_WIZARD"`
	// Provisioner defines the type of provisioner to use
	Provisioner provisionerType `json:"provisioner" yaml:"provisioner" env:"ROBO_WIZARD"`
	// DumpCore defines a command to collect all installation/operation logs
	DumpCore bool `json:"-" yaml:"-"`
	// StateDir specifies the location for test-specific temporary data
	StateDir string `json:"-" yaml:"-"`
	// Teardown specifies if the cluster should be destroyed at the end of this
	// test run
	Teardown bool `json:"-" yaml:"-"`
	// ReportDir defines location to store the results of the test
	ReportDir string `json:"report_dir" yaml:"report_dir" env:"ROBO_REPORT_DIR"`
	// ClusterName defines the name to use for domain name or state directory
	ClusterName string `json:"cluster_name" yaml:"cluster_name" env:"ROBO_CLUSTER_NAME"`
	// License specifies the application license
	License string `json:"license" yaml:"license" env:"ROBO_APP_LICENSE"`
	// OpsCenterURL defines the URL of the existing Ops Center.
	// This specifies the original Ops Center for the wizard test flow
	// and will be used to upload application updates.
	// This is a requirement for all browser-based tests
	OpsCenterURL string `json:"ops_url" yaml:"ops_url" env:"ROBO_OPS_URL"`
	// Application defines the application package to test
	Application *loc.Locator `json:"application" yaml:"application" env:"ROBO_APP"`
	// Login defines the login details to access the Ops Center
	Login Login `json:"login" yaml:"login"`
	// ServiceLogin defines the login parameters for service access to the Ops Center
	ServiceLogin ServiceLogin `json:"service_login" yaml:"service_login"`
	// FlavorLabel specifies the installation flavor label to use for the test.
	// This is application-specific, e.g. `3 nodes` or `medium`
	FlavorLabel string `json:"flavor_label" yaml:"flavor_label" env:"ROBO_FLAVOR_LABEL"`
	// AWS defines the AWS-specific test configuration
	AWS AWSConfig `json:"aws" yaml:"aws"`
	// Onprem defines the test configuration for bare metal tests
	Onprem OnpremConfig `json:"onprem" yaml:"onprem"`
}

// Login defines Ops Center authentication parameters
type Login struct {
	Username string `json:"username" yaml:"username" env:"ROBO_USERNAME"`
	Password string `json:"password" yaml:"password" env:"ROBO_PASSWORD"`
	// AuthProvider specifies the authentication provider to use for login.
	// Available providers are `email` and `gogole`
	AuthProvider string `json:"auth_provider" yaml:"auth_provider" env:"ROBO_AUTH_PROVIDER"`
}

func (r Login) IsEmpty() bool {
	return r.Username == "" && r.Password == ""
}

// ServiceLogin defines authentication options for Ops Center service access
type ServiceLogin struct {
	Username string `json:"username" yaml:"username" env:"ROBO_SERVICE_USERNAME"`
	Password string `json:"password" yaml:"password" env:"ROBO_SERVICE_PASSWORD"`
}

func (r ServiceLogin) IsEmpty() bool {
	return r.Username == "" && r.Password == ""
}

// AWSConfig describes AWS EC2 test configuration
type AWSConfig struct {
	AccessKey string `json:"access_key" yaml:"access_key" env:"ROBO_AWS_ACCESS_KEY"`
	SecretKey string `json:"secret_key" yaml:"secret_key" env:"ROBO_AWS_SECRET_KEY"`
	// Region specifies the EC2 region to install into
	Region string `json:"region" yaml:"region" env:"ROBO_AWS_REGION"`
	// KeyPair specifies the name of the SSH key pair to use for provisioning
	// nodes
	KeyPair string `json:"key_pair" yaml:"key_pair" env:"ROBO_AWS_KEY_PAIR"`
	// VPC defines the Amazon VPC to install into.
	// Specify "Create new" to create a new VPC for this test run
	VPC string `json:"vpc" yaml:"vpc" env:"ROBO_AWS_VPC"`
	// KeyPath specifies the location of the SSH key to use for remote access.
	// Mandatory only with terraform provisioner
	KeyPath string `json:"key_path" yaml:"key_path" env:"ROBO_AWS_KEY_PATH"`
	// InstanceType defines the type of AWS EC2 instance to boot.
	// Relevant only with terraform provisioner.
	// Defaults are specific to the terraform script used (if any)
	InstanceType string `json:"instance_type" yaml:"instance_type" env:"ROBO_AWS_INSTANCE_TYPE"`
}

func (r AWSConfig) IsEmpty() bool {
	return r.AccessKey == "" && r.SecretKey == ""
}

// OnpremConfig defines the test configuration for bare metal tests
type OnpremConfig struct {
	// NumNodes defines the total cluster capacity.
	// This is a total number of nodes to provision
	NumNodes int `json:"nodes" yaml:"nodes" env:"ROBO_NUM_NODES"`
	// InstallerURL defines the location of the installer tarball.
	// Depending on the provisioner - this can be either a URL or local path
	InstallerURL string `json:"installer_url" yaml:"installer_url" env:"ROBO_INSTALLER_URL"`
	// ScriptPath defines the path to the provisioner script.
	// TODO: if unspecified, scripts in assets/<provisioner> are used
	ScriptPath string `json:"script_path" yaml:"script_path" env:"ROBO_SCRIPT_PATH"`
}

func (r OnpremConfig) IsEmpty() bool {
	return r.NumNodes == 0 && r.InstallerURL == "" && r.ScriptPath == ""
}

func registerCommonFlags() {
	// Turn on verbose by default to get spec names
	config.DefaultReporterConfig.Verbose = true
	// Turn on EmitSpecProgress to get spec progress (especially on interrupt)
	config.GinkgoConfig.EmitSpecProgress = true

	flag.StringVar(&configFile, "config", "config.yaml", "Configuration file to use")
	flag.StringVar(&stateConfigFile, "state-file", "config.yaml.state", "State configuration file to use")
	flag.BoolVar(&debugFlag, "debug", false, "Verbose mode")
	flag.Var(&mode, "mode", "Run tests in specific mode. Supported modes: [`wizard`]")
	flag.BoolVar(&teardownFlag, "destroy", false, "Destroy infrastructure after all tests")
	flag.BoolVar(&dumpFlag, "report", false, "Collect installation and operation logs into the report directory")
	flag.StringVar(&provisionerName, "provisioner", "", "Provision nodes using this provisioner")
}

func newFileConfig(input io.Reader) error {
	configBytes, err := ioutil.ReadAll(input)
	if err != nil {
		return trace.Wrap(err)
	}
	err = yaml.Unmarshal(configBytes, &TestContext)
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
			Config:       infraConfig,
			ScriptPath:   TestContext.Onprem.ScriptPath,
			InstallerURL: TestContext.Onprem.InstallerURL,
			NumNodes:     TestContext.Onprem.NumNodes,
			AccessKey:    TestContext.AWS.AccessKey,
			SecretKey:    TestContext.AWS.SecretKey,
			KeyPair:      TestContext.AWS.KeyPair,
			SSHKeyPath:   TestContext.AWS.KeyPath,
			InstanceType: TestContext.AWS.InstanceType,
		}
		err := config.Validate()
		if err != nil {
			return nil, trace.Wrap(err)
		}
		provisioner, err = terraform.New(stateDir, config)
	case provisionerVagrant:
		config := vagrant.Config{
			Config:       infraConfig,
			ScriptPath:   TestContext.Onprem.ScriptPath,
			InstallerURL: TestContext.Onprem.InstallerURL,
			NumNodes:     TestContext.Onprem.NumNodes,
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
			Config:       infraConfig,
			ScriptPath:   TestContext.Onprem.ScriptPath,
			InstallerURL: TestContext.Onprem.InstallerURL,
			NumNodes:     len(testState.ProvisionerState.Nodes),
			AccessKey:    TestContext.AWS.AccessKey,
			SecretKey:    TestContext.AWS.SecretKey,
			KeyPair:      TestContext.AWS.KeyPair,
			SSHKeyPath:   TestContext.AWS.KeyPath,
			InstanceType: TestContext.AWS.InstanceType,
		}
		err := config.Validate()
		if err != nil {
			return nil, trace.Wrap(err)
		}
		provisioner, err = terraform.NewFromState(config, *testState.ProvisionerState)
	case provisionerVagrant:
		config := vagrant.Config{
			Config:       infraConfig,
			ScriptPath:   TestContext.Onprem.ScriptPath,
			InstallerURL: TestContext.Onprem.InstallerURL,
			NumNodes:     len(testState.ProvisionerState.Nodes),
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

func (r *modeType) String() string {
	return string(*r)
}

func (r *modeType) Set(value string) error {
	*r = modeType(value)
	if *r == "" {
		*r = wizardMode
	}
	return nil
}

type modeType string

const (
	wizardMode modeType = "wizard"
)

// configFile defines the configuration file to use for the tests
var configFile string

// stateConfigFile defines the state configuration file to use for the tests
var stateConfigFile string

// debugFlag defines whether to run in verbose mode
var debugFlag bool

// mode defines the mode for tests
var mode modeType

// provisionerName defines the provisioner to use to provision nodes in the test cluster
var provisionerName string

// teardownFlag defines if the cluster should be destroyed
var teardownFlag bool

// dumpFlag defines whether to collect installation and operation logs
var dumpFlag bool
