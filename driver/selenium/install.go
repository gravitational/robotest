package selenium

import (
	"fmt"
	"strings"
	"time"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/lib/defaults"
	"github.com/gravitational/robotest/lib/wait"
	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
	"github.com/tebeka/selenium"
)

func (r *driver) Install(cluster infra.Infra, installerURL string) error {
	log.Infof("opening %v", installerURL)
	err := r.Get(installerURL)
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(actionDelay)

	err = processLicenseScreen(r.WebDriver, r.License)
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(actionDelay)

	err = processNewSiteScreen(r.WebDriver, r.ClusterName)
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(actionDelay)

	err = processCapacityScreen(r.WebDriver, r.FlavorLabel, cluster)
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(actionDelay)

	err = processInstallResult(r.WebDriver)
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(actionDelay)

	err = processBandwagonScreen(r.WebDriver)
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(actionDelay)

	return nil
}

// processLicenseScreen enters a license if it needs to be entered and proceeds to the next screen
func processLicenseScreen(wd selenium.WebDriver, license string) error {
	// see if license is required
	elem, _ := wd.FindElement(selenium.ByCSSSelector, ".grv-installer-license")
	if elem == nil {
		// log.Infof("%v does not need a license", r.conf.Application)
		return nil
	}

	// enter a license
	log.Infof("entering license")
	err := wait.Retry(defaults.RetryDelay, defaults.RetryAttempts, func() error {
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
		time.Sleep(actionDelay)
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
	err := wait.Retry(defaults.RetryDelay, defaults.RetryAttempts, func() error {
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

	time.Sleep(actionDelay)

	// select onprem install type
	log.Infof("selecting onprem installation type")
	err = wait.Retry(defaults.RetryDelay, defaults.RetryAttempts, func() error {
		elem, err := wd.FindElement(selenium.ByCSSSelector, ".--metal")
		if err != nil {
			return trace.Wrap(err)
		}
		return trace.Wrap(elem.Click())
	})
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(actionDelay)

	// click "next"
	log.Infof("creating a site")
	err = wait.Retry(defaults.RetryDelay, defaults.RetryAttempts, func() error {
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
func processCapacityScreen(wd selenium.WebDriver, flavorLabel string, cluster infra.Infra) error {
	// select flavor
	log.Infof("selecting flavor with label %q", flavorLabel)
	err := wait.Retry(defaults.RetryDelay, defaults.RetryAttempts, func() error {
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

	time.Sleep(actionDelay)

	// find curl command for gravity agent
	log.Infof("extracting agent command")
	var command string
	err = wait.Retry(defaults.RetryDelay, defaults.RetryAttempts, func() error {
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

	time.Sleep(actionDelay)

	log.Infof("running agent command %q on %v", command, cluster)
	errCh := make(chan error, 1)
	go func() {
		errCh <- infra.Distribute(command, cluster.Provisioner().Nodes())
		close(errCh)
	}()

	log.Infof("waiting for agents to register")
	ticker := time.NewTicker(defaults.RetryDelay)
	var elems []selenium.WebElement
L:
	for i := 0; i < defaults.RetryAttempts; i++ {
		select {
		case err := <-errCh:
			if err != nil {
				return trace.Wrap(err)
			}
			errCh = nil
		case <-ticker.C:
			elems, err = wd.FindElements(selenium.ByCSSSelector, ".grv-provision-req-server-inputs")
			if err != nil {
				return trace.Wrap(err)
			}
			// FIXME: Cluster.NumNodes should return the number of active nodes
			if len(elems) == cluster.Provisioner().NumNodes() {
				break L
			}
		}
	}
	ticker.Stop()

	if len(elems) != cluster.Provisioner().NumNodes() {
		return trace.NotFound("timed out waiting for agents: got %d, want %d", len(elems), cluster.Provisioner().NumNodes())
	}
	log.Infof("all agents have joined")
	time.Sleep(actionDelay)

	// click "start installation"
	log.Infof("starting installation")
	err = wait.Retry(defaults.RetryDelay, defaults.RetryAttempts, func() error {
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
	err := wait.Retry(installRetryDelay, installRetryAttempts, func() error {
		page, err := wd.PageSource()
		if err != nil {
			return trace.Wrap(err)
		}
		if strings.Contains(page, "Installation Successful") {
			return nil
		}
		if strings.Contains(page, "Install failure") {
			return wait.Abort(trace.Errorf("install failure"))
		}
		return wait.Continue("waiting for the installation to complete...")
	})
	if err != nil {
		return trace.Wrap(err)
	}

	log.Infof("installation success!")
	time.Sleep(actionDelay)

	// click "continue" button
	log.Infof("proceed to final install step")
	err = wait.Retry(defaults.RetryDelay, defaults.RetryAttempts, func() error {
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
	err := wait.Retry(defaults.RetryDelay, defaults.RetryAttempts, func() error {
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
		time.Sleep(actionDelay)
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
		time.Sleep(actionDelay)
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
		time.Sleep(actionDelay)
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
		time.Sleep(actionDelay)
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

	time.Sleep(actionDelay)

	// click "finish"
	err = wait.Retry(defaults.RetryDelay, defaults.RetryAttempts, func() error {
		elem, err := wd.FindElement(selenium.ByCSSSelector, ".my-page-btn-submit")
		if err != nil {
			return trace.Wrap(err)
		}
		return trace.Wrap(elem.Click())
	})
	return trace.Wrap(err)
}
