package framework

import (
	"fmt"
	"net/url"

	"github.com/onsi/gomega"
)

func SiteURL() string {
	URL, err := url.Parse(Cluster.OpsCenterURL())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	URL.RawQuery = ""
	URL.Path = fmt.Sprintf("web/site/%v", TestContext.ClusterName)
	return URL.String()
}

func InstallerURL() string {
	installerURL, err := url.Parse(Cluster.OpsCenterURL())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	installerURL.RawQuery = ""
	installerURL.Path = fmt.Sprintf("web/installer/new/%v/%v/%v",
		TestContext.Application.Repository, TestContext.Application.Name, TestContext.Application.Version)
	return installerURL.String()
}
