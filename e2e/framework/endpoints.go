package framework

import (
	"fmt"
	"net/url"

	"github.com/gravitational/trace"
)

func InstallerURL() (string, error) {
	installerURL, err := url.Parse(Cluster.OpsCenterURL())
	if err != nil {
		return "", trace.Wrap(err)
	}
	installerURL.RawQuery = ""
	installerURL.Path = fmt.Sprintf("web/installer/new/%v/%v/%v",
		TestContext.Application.Repository, TestContext.Application.Name, TestContext.Application.Version)
	return installerURL.String(), nil
}
