package framework

import (
	"fmt"
	"net/url"

	"github.com/gravitational/robotest/lib/loc"
	"github.com/gravitational/trace"
)

func InstallerURL(opsCenterURL string, appPackage loc.Locator) (string, error) {
	url, err := url.Parse(opsCenterURL)
	if err != nil {
		return "", trace.Wrap(err)
	}
	url.Path = fmt.Sprintf("web/installer/new/%v/%v/%v",
		appPackage.Repository, appPackage.Name, appPackage.Version)
	return url.String(), nil
}
