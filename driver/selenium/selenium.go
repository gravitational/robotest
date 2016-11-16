package selenium

import (
	"time"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/trace"

	"github.com/tebeka/selenium"
)

func New(config Config) (*driver, error) {
	err := config.Validate()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	caps := selenium.Capabilities{"browserName": config.BrowserName}
	remote, err := selenium.NewRemote(caps, "")
	if err != nil {
		return nil, trace.Wrap(err)
	}

	remote.SetImplicitWaitTimeout(driverWaitTimeout)

	return &driver{
		WebDriver: remote,
		Config:    config,
	}, nil
}

func (r *driver) Close() error {
	return r.WebDriver.Close()
}

func (r *Config) Validate() error {
	err := r.Config.Validate()
	if err != nil {
		return trace.Wrap(err)
	}
	if r.FlavorLabel == "" {
		return trace.BadParameter("install flavor label is required")
	}
	if r.BrowserName == "" {
		r.BrowserName = "chrome"
	}
	return nil
}

type driver struct {
	selenium.WebDriver
	Config
}

type Config struct {
	infra.Config
	// FlavorLabel defines the flavor to install using the flavor label
	FlavorLabel string `json:"install_flavor_label" env:"ROBO_FLAVOR_LABEL"`
	// BrowserName defines the browser to use
	BrowserName string `json:"browser_name" env:"ROBO_BROWSER_NAME"`
}

const (
	// actionDelay is the delay between selenium actions
	actionDelay = 2 * time.Second

	// installRetryDelay defines the interval between retry attempts during installation
	installRetryDelay = 10 * time.Second
	// installRetryAttempts defines the maximum number of retry attempts for installation
	installRetryAttempts = 200

	driverWaitTimeout = 1 * time.Second
)
