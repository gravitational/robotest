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

	confFile, err := os.Open(configFile)
	if err != nil {
		Failf("failed to read configuration from %q", configFile)
	}
	defer confFile.Close()
	err = newFileConfig(confFile)
	if err != nil {
		Failf("failed to read configuration from %q", configFile)
	}
	if wizardFlag {
		TestContext.Wizard = wizardFlag
	}
	if provisionerName != "" {
		TestContext.Provisioner = provisionerName
	}

	initLogger(debugFlag)
}

func (r *TestContextType) Validate() error {
	var errors []error
	if TestContext.Wizard && TestContext.Onprem.InstallerURL == "" {
		errors = append(errors, trace.BadParameter("installer URL is required in wizard mode"))
	}
	if TestContext.AWS == nil && TestContext.Onprem == nil {
		errors = append(errors, trace.BadParameter("either AWS or Onprem is required"))
	}
	if r.Onprem != nil && r.NumInstallNodes > r.Onprem.NumNodes {
		errors = append(errors, trace.BadParameter("cannot install on more nodes than the cluster capacity: %v > %v",
			r.NumInstallNodes, r.Onprem.NumNodes))
	}
	return trace.NewAggregate(errors...)
}

func Failf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Info(msg)
	ginkgo.Fail(nowStamp()+": "+msg, 1)
}

// TestContext defines the global test configuration for the test run
var TestContext TestContextType

type TestContextType struct {
	Wizard      bool   `json:"wizard" env:"ROBO_WIZARD"`
	Provisioner string `json:"provisioner" env:"ROBO_WIZARD"`
	// ReportDir defines location to store the results of the test
	ReportDir string `json:"report_dir" env:"ROBO_REPORT_DIR"`
	// ClusterName defines the name to use for domain name or state directory
	ClusterName string `json:"cluster_name" env:"ROBO_CLUSTER_NAME"`
	// OpsCenterURL defines the URL of the existing OpsCenter.
	// This is a requirement for all browser-based tests
	OpsCenterURL string `json:"ops_url" env:"ROBO_CLUSTER_NAME"`
	// Application defines the application package to test
	Application *loc.Locator `json:"application" env:"ROBO_APP"`
	// Login defines the login details to access the OpsCenter
	Login Login `json:"login"`
	// NumInstallNodes defines the subset of nodes to use for installation.
	NumInstallNodes int `json:"install_nodes" env:"ROBO_NUM_INSTALL_NODES"`
	// AWS defines the AWS-specific test configuration
	AWS *AWSConfig `json:"aws"`
	// Onprem defines the test configuration for bare metal tests
	Onprem *OnpremConfig `json:"onprem"`
}

type Login struct {
	Username string `json:"username" env:"ROBO_USERNAME"`
	Password string `json:"password" env:"ROBO_PASSWORD"`
}

type AWSConfig struct {
	AccessKey string `json:"access_key" env:"ROBO_AWS_ACCESS_KEY"`
	SecretKey string `json:"secret_key" env:"ROBO_AWS_SECRET_KEY"`
	Region    string `json:"region" env:"ROBO_AWS_REGION"`
	KeyPair   string `json:"key_pair" env:"ROBO_AWS_KEYPAIR"`
	VPC       string `json:"vpc" env:"ROBO_AWS_VPC"`
}

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

func registerCommonFlags() {
	// Turn on verbose by default to get spec names
	config.DefaultReporterConfig.Verbose = true
	// Turn on EmitSpecProgress to get spec progress (especially on interrupt)
	config.GinkgoConfig.EmitSpecProgress = true

	flag.StringVar(&configFile, "config-file", "config.json", "Configuration file to use")
	flag.BoolVar(&debugFlag, "debug", false, "Verbose mode")
	flag.BoolVar(&wizardFlag, "wizard", false, "Run tests in wizard mode")
	flag.StringVar(&provisionerName, "provisioner", "", "Provision nodes using this provisioner")
}

func newFileConfig(input io.Reader) error {
	d := json.NewDecoder(input)
	err := d.Decode(&TestContext)
	if err != nil {
		return trace.Wrap(err)
	}

	err = configure.ParseEnv(&TestContext)
	if err != nil {
		return trace.Wrap(err)
	}

	err = TestContext.Validate()
	if err != nil {
		return trace.Wrap(err, "failed to validate configuration")
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

func provisionerFromConfig(infraConfig infra.Config, stateDir, provisionerName string) (provisioner infra.Provisioner, err error) {
	switch provisionerName {
	case "terraform":
		config := terraform.Config{
			Config: infraConfig,
		}
		provisioner, err = terraform.New(stateDir, config)
	case "vagrant":
		config := vagrant.Config{
			Config:          infraConfig,
			ScriptPath:      TestContext.Onprem.ScriptPath,
			InstallerURL:    TestContext.Onprem.InstallerURL,
			NumNodes:        TestContext.Onprem.NumNodes,
			NumInstallNodes: TestContext.NumInstallNodes,
		}
		provisioner, err = vagrant.New(stateDir, config)
	default:
		// no provisioner when the cluster has already been provisioned
		// or automatic provisioning is used
	}
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return provisioner, nil
}

// configFile defines the configuration file to use for the tests
var configFile string

// debugFlag defines whether to run in verbose mode
var debugFlag bool

// wizardFlag defines whether to run tests in wizard mode
var wizardFlag bool

// provisionerNAme defines the provisioner to use to provision nodes in the test cluster
var provisionerName string
