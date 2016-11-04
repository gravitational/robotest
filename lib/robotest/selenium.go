package robotest

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/trace"
	"github.com/tebeka/selenium"
)

func runSelenium(url string, conf config, tf terraformOutput) error {
	caps := selenium.Capabilities{"browserName": "chrome"}
	wd, err := selenium.NewRemote(caps, "")
	if err != nil {
		return trace.Wrap(err)
	}
	defer wd.Close()

	wd.SetImplicitWaitTimeout(5 * time.Second)

	log.Infof("opening %v", url)
	err = wd.Get(url)
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(seleniumDelay)

	err = processLicenseScreen(wd, conf.LicensePath)
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(seleniumDelay)

	err = processNewSiteScreen(wd, conf.ClusterName)
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(seleniumDelay)

	err = processCapacityScreen(wd, conf.FlavorLabel, tf.publicIPs, conf.SSHKeyPath)
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(seleniumDelay)

	err = processInstallResult(wd)
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(seleniumDelay)

	err = processBandwagonScreen(wd)
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(seleniumDelay)

	return nil
}

// processLicenseScreen enters a license if it needs to be entered and proceeds to the next screen
func processLicenseScreen(wd selenium.WebDriver, licensePath string) error {
	// see if license is required
	elem, _ := wd.FindElement(selenium.ByCSSSelector, ".grv-installer-license")
	if elem == nil {
		log.Infof("app does not need a license")
		return nil
	}

	license, err := ioutil.ReadFile(licensePath)
	if err != nil {
		return trace.Wrap(err)
	}

	// enter a license
	log.Infof("entering license")
	err = utils.Retry(defaultRetryDelay, defaultRetryAttempts, func() error {
		textarea, err := elem.FindElement(selenium.ByTagName, "textarea")
		if err != nil {
			return trace.Wrap(err)
		}
		err = textarea.Clear()
		if err != nil {
			return trace.Wrap(err)
		}
		err = textarea.SendKeys(string(license))
		if err != nil {
			return trace.Wrap(err)
		}
		time.Sleep(seleniumDelay)
		elem, err := wd.FindElement(selenium.ByCSSSelector, ".grv-installer-btn-new-site")
		if err != nil {
			return trace.Wrap(err)
		}
		return trace.Wrap(elem.Click())
	})
	return trace.Wrap(err)
}

// processNewSiteScreen enters cluster name, selects on-prem install type and proceeds
// to the next screen
func processNewSiteScreen(wd selenium.WebDriver, clusterName string) error {
	// enter domain name
	log.Infof("entering domain name")
	err := utils.Retry(defaultRetryDelay, defaultRetryAttempts, func() error {
		elem, err := wd.FindElement(selenium.ByName, "domainName")
		if err != nil {
			return trace.Wrap(err)
		}
		err = elem.Clear()
		if err != nil {
			return trace.Wrap(err)
		}
		return trace.Wrap(elem.SendKeys(clusterName))
	})
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(seleniumDelay)

	// select onprem install type
	log.Infof("selecting onprem installation type")
	err = utils.Retry(defaultRetryDelay, defaultRetryAttempts, func() error {
		elem, err := wd.FindElement(selenium.ByCSSSelector, ".--metal")
		if err != nil {
			return trace.Wrap(err)
		}
		return trace.Wrap(elem.Click())
	})
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(seleniumDelay)

	// click "next"
	log.Infof("creating a site")
	err = utils.Retry(defaultRetryDelay, defaultRetryAttempts, func() error {
		elem, err := wd.FindElement(selenium.ByCSSSelector, ".grv-installer-btn-new-site")
		if err != nil {
			return trace.Wrap(err)
		}
		return trace.Wrap(elem.Click())
	})
	return trace.Wrap(err)
}

// processCapacityScreen selects a requested flavor, runs agent command, waits for all agents to connect
// and launches the installation
func processCapacityScreen(wd selenium.WebDriver, flavorLabel string, ips []string, keyPath string) error {
	// select flavor
	log.Infof("selecting flavor with label '%v'", flavorLabel)
	err := utils.Retry(defaultRetryDelay, defaultRetryAttempts, func() error {
		elems, err := wd.FindElements(selenium.ByCSSSelector, ".grv-slider-value-desc")
		if err != nil {
			return trace.Wrap(err)
		}
		for _, elem := range elems {
			span, err := elem.FindElement(selenium.ByTagName, "span")
			if err != nil {
				return trace.Wrap(err)
			}
			text, err := span.Text()
			if err != nil {
				return trace.Wrap(err)
			}
			if text == flavorLabel {
				return trace.Wrap(span.Click())
			}
		}
		return trace.BadParameter("could not find flavor")
	})
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(seleniumDelay)

	// find curl command for gravity agent
	log.Infof("extracting agent command")
	var command string
	err = utils.Retry(defaultRetryDelay, defaultRetryAttempts, func() error {
		elem, err := wd.FindElement(selenium.ByCSSSelector, ".m-t-sm")
		if err != nil {
			return trace.Wrap(err)
		}
		elem, err = elem.FindElement(selenium.ByTagName, "span")
		if err != nil {
			return trace.Wrap(err)
		}
		command, err = elem.Text()
		return trace.Wrap(err)
	})
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(seleniumDelay)

	log.Infof("running agent command on servers: %v", command)
	for _, ip := range ips {
		go startAgent(ip, command, keyPath)
	}

	// wait for agents to register
	log.Infof("waiting for agents to register")
	err = utils.Retry(defaultRetryDelay, defaultRetryAttempts, func() error {
		elems, err := wd.FindElements(selenium.ByCSSSelector, ".grv-provision-req-server-inputs")
		if err != nil {
			return trace.Wrap(err)
		}
		if len(elems) != len(ips) {
			return trace.BadParameter("not all agents have joined yet")
		}
		return nil
	})
	if err != nil {
		return trace.Wrap(err)
	}

	log.Infof("all agents have joined")
	time.Sleep(seleniumDelay)

	// click "start installation"
	log.Infof("starting installation")
	err = utils.Retry(defaultRetryDelay, defaultRetryAttempts, func() error {
		elem, err := wd.FindElement(selenium.ByCSSSelector, ".grv-installer-btn-new-site")
		if err != nil {
			return trace.Wrap(err)
		}
		return trace.Wrap(elem.Click())
	})
	return trace.Wrap(err)
}

// processInstallResult waits for the install result and proceeds to the next screen if it was
// successful
func processInstallResult(wd selenium.WebDriver) error {
	log.Infof("waiting for the installation to complete")
	err := utils.Retry(installRetryDelay, installRetryAttempts, func() error {
		page, err := wd.PageSource()
		if err != nil {
			return trace.Wrap(err)
		}
		if strings.Contains(page, "Installation Successful") {
			return nil
		}
		if strings.Contains(page, "Install failure") {
			return utils.Abort(trace.Errorf("install failure"))
		}
		return utils.Continue("waiting for the installation to complete...")
	})
	if err != nil {
		return trace.Wrap(err)
	}

	log.Infof("installation success!")
	time.Sleep(seleniumDelay)

	// click "continue" button
	log.Infof("proceed to final install step")
	err = utils.Retry(defaultRetryDelay, defaultRetryAttempts, func() error {
		elem, err := wd.FindElement(selenium.ByCSSSelector, ".grv-installer-progress-result")
		if err != nil {
			return trace.Wrap(err)
		}
		elem, err = elem.FindElement(selenium.ByTagName, "a")
		if err != nil {
			return trace.Wrap(err)
		}
		return trace.Wrap(elem.Click())
	})
	return trace.Wrap(err)
}

// processBandwagonScreen fills out all bandwagon fields and proceeds to the next screen
func processBandwagonScreen(wd selenium.WebDriver) error {
	log.Infof("filling out bandwagon fields")
	err := utils.Retry(defaultRetryDelay, defaultRetryAttempts, func() error {
		elem, err := wd.FindElement(selenium.ByName, "email")
		if err != nil {
			return trace.Wrap(err)
		}
		err = elem.Clear()
		if err != nil {
			return trace.Wrap(err)
		}
		err = elem.SendKeys(fmt.Sprintf("robotest@example.com"))
		if err != nil {
			return trace.Wrap(err)
		}
		time.Sleep(seleniumDelay)
		elem, err = wd.FindElement(selenium.ByName, "password")
		if err != nil {
			return trace.Wrap(err)
		}
		err = elem.Clear()
		if err != nil {
			return trace.Wrap(err)
		}
		err = elem.SendKeys("Robotest_123")
		if err != nil {
			return trace.Wrap(err)
		}
		time.Sleep(seleniumDelay)
		elem, err = wd.FindElement(selenium.ByName, "passwordConfirmed")
		if err != nil {
			return trace.Wrap(err)
		}
		err = elem.Clear()
		if err != nil {
			return trace.Wrap(err)
		}
		err = elem.SendKeys("Robotest_123")
		if err != nil {
			return trace.Wrap(err)
		}
		time.Sleep(seleniumDelay)
		// the following elements are optional
		elem, _ = wd.FindElement(selenium.ByName, "org")
		if elem != nil {
			err = elem.Clear()
			if err != nil {
				return trace.Wrap(err)
			}
			err = elem.SendKeys("Organization")
			if err != nil {
				return trace.Wrap(err)
			}
		}
		time.Sleep(seleniumDelay)
		elem, _ = wd.FindElement(selenium.ByName, "name")
		if elem != nil {
			err = elem.Clear()
			if err != nil {
				return trace.Wrap(err)
			}
			err = elem.SendKeys("robotest")
			if err != nil {
				return trace.Wrap(err)
			}
		}
		return nil
	})
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(seleniumDelay)

	// click "finish"
	err = utils.Retry(defaultRetryDelay, defaultRetryAttempts, func() error {
		elem, err := wd.FindElement(selenium.ByCSSSelector, ".my-page-btn-submit")
		if err != nil {
			return trace.Wrap(err)
		}
		return trace.Wrap(elem.Click())
	})
	return trace.Wrap(err)
}

const (
	// seleniumDelay is the delay between selenium actions
	seleniumDelay = 2 * time.Second

	// defaultRetry(Delay|Attempts) control delay/attempts for non-install actions
	defaultRetryDelay    = 5 * time.Second
	defaultRetryAttempts = 100

	// installRetry(Delay|Attempts) control delay/attempts for the installation
	installRetryDelay    = 10 * time.Second
	installRetryAttempts = 200
)
