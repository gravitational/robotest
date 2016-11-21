package framework

import (
	"fmt"
	"net/url"

	"github.com/onsi/gomega"
)

func InstallerURL() string {
	installerURL, err := url.Parse(Cluster.OpsCenterURL())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	installerURL.RawQuery = ""
	installerURL.Path = fmt.Sprintf("web/installer/new/%v/%v/%v",
		TestContext.Application.Repository, TestContext.Application.Name, TestContext.Application.Version)
	return installerURL.String()
}
