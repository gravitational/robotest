package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gravitational/robotest/driver/selenium"
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/terraform"
	"github.com/gravitational/robotest/infra/vagrant"
	debugutils "github.com/gravitational/robotest/lib/debug"
	"github.com/gravitational/robotest/lib/system"

	"github.com/gravitational/configure/cstrings"
	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	if err := run(); err != nil {
		log.Errorf(trace.DebugReport(err))
		os.Exit(255)
	}
}

func run() error {
	args, _ := cstrings.SplitAt(os.Args, "--")

	var (
		app        = kingpin.New("robotest", "Automatic gravity tests")
		debug      = app.Flag("debug", "Enable debug mode").Bool()
		configS    = app.Flag("config", "JSON string with configuration").Default("{}").String()
		configFile = app.Flag("config-file", "path to configuration file").String()

		crun = app.Command("run", "Execute tests")
	)

	cmd, err := app.Parse(args[1:])
	if err != nil {
		return trace.Wrap(err)
	}

	if *debug {
		initLogger(log.DebugLevel)
		go debugutils.DumpLoop()
	}

	var configReader io.ReadCloser
	if *configS != "{}" {
		configReader = ioutil.NopCloser(strings.NewReader(*configS))
	} else {
		configReader, err = os.Open(*configFile)
		if err != nil {
			return trace.Wrap(err)
		}
	}
	defer configReader.Close()

	config, err := newFileConfig(configReader)
	if err != nil {
		return trace.Wrap(err)
	}

	dir, err := stateDir(config.ClusterName)
	if err != nil {
		return trace.Wrap(err)
	}

	provisioner, err := provisionerFromConfig(*config, dir)
	if err != nil {
		return trace.Wrap(err)
	}

	cluster, output, err := infra.NewWizard(config.Config, provisioner)
	if err != nil {
		return trace.Wrap(err)
	}
	defer cluster.Close()

	driver, err := driverFromConfig(*config)
	if err != nil {
		return trace.Wrap(err)
	}

	switch cmd {
	case crun.FullCommand():
		err = runTests(*config, cluster, *output, driver)
	}

	if err == nil {
		errDelete := system.RemoveContents(dir)
		if errDelete != nil {
			log.Errorf("failed to delete %q: %v", dir, err)
		}
	}

	return trace.Wrap(err)
}

func runTests(config fileConfig, cluster infra.Infra, output infra.ProvisionerOutput, driver infra.TestDriver) error {
	err := driver.Install(cluster, output.InstallerURL.String())
	if err != nil {
		return trace.Wrap(err)
	}
	return nil
}

func provisionerFromConfig(config fileConfig, dir string) (provisioner infra.Provisioner, err error) {
	switch {
	case config.Provisioner.Terraform != nil:
		config.Provisioner.Terraform.Config = config.Config
		provisioner, err = terraform.New(dir, *config.Provisioner.Terraform)
	case config.Provisioner.Vagrant != nil:
		config.Provisioner.Vagrant.Config = config.Config
		provisioner, err = vagrant.New(dir, *config.Provisioner.Vagrant)
	default:
		return nil, trace.BadParameter("no provisioner enabled in configuration")
	}
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return provisioner, nil
}

func driverFromConfig(config fileConfig) (driver infra.TestDriver, err error) {
	switch {
	case config.Driver.Web != nil:
		config.Driver.Web.Config = config.Config
		driver, err = selenium.New(*config.Driver.Web)
	default:
		return nil, trace.BadParameter("no driver enabled in configuration")
	}
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return driver, nil
}

func initLogger(level log.Level) {
	log.StandardLogger().Hooks = make(log.LevelHooks)
	log.SetFormatter(&trace.TextFormatter{TextFormatter: log.TextFormatter{FullTimestamp: true}})
	log.SetOutput(os.Stderr)
	log.SetLevel(level)
}

func stateDir(clusterName string) (dir string, err error) {
	dir, err = ioutil.TempDir("", fmt.Sprintf("robotest-%v-", clusterName))
	if err != nil {
		return "", trace.Wrap(err)
	}
	log.Infof("state dir: %v", dir)
	return dir, nil
}
