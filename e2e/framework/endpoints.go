package framework

import (
	"fmt"
	"net/url"

	"github.com/onsi/gomega"
)

// SiteURL returns URL of the site specified in configuration as TestContext.ClusterName
func SiteURL() string {
	path := fmt.Sprintf("web/site/%v", TestContext.ClusterName)
	return URLPath(path)
}

// InstallerURL returns URL of the installer for the configured application package
func InstallerURL() string {
	path := fmt.Sprintf("web/installer/new/%v/%v/%v",
		TestContext.Application.Repository, TestContext.Application.Name, TestContext.Application.Version)
	return URLPath(path)
}

// URLPath returns a new URL with the configured Ops Center using path as a custom URL path
func URLPath(path string) string {
	url, err := url.Parse(Cluster.OpsCenterURL())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	url.RawQuery = ""
	url.Path = path
	return url.String()
}

// URLPathFromString returns a new URL with the specified URL urlS using path as a custom URL path
func URLPathFromString(urlS string, path string) string {
	url, err := url.Parse(urlS)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	url.RawQuery = ""
	url.Path = path
	return url.String()
}
