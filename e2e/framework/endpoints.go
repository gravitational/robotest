/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	gomega.Expect(TestContext.Application.Locator).NotTo(gomega.BeNil(), "should have a valid application package")

	path := fmt.Sprintf("web/installer/new/%v/%v/%v",
		TestContext.Application.Repository, TestContext.Application.Name, TestContext.Application.Version)
	return URLPath(path)
}

// SiteInstallURL returns URL of current cluster installation
func SiteInstallURL(clusterName string) string {
	path := fmt.Sprintf("web/installer/site/%v/", clusterName)
	return URLPath(path)
}

// URLPath returns a new URL from the configured entry URL using path as new URL path
func URLPath(path string) string {
	url, err := url.Parse(TestContext.OpsCenterURL)
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

// UpdateSiteEntry specifies new entryURL and login details
// to use for subsequent site access.
func UpdateSiteEntry(entryURL string, login Login, serviceLogin *ServiceLogin) {
	TestContext.OpsCenterURL = entryURL
	TestContext.Login = login
	if serviceLogin != nil {
		TestContext.ServiceLogin = *serviceLogin
	}
	testState.EntryURL = entryURL
	testState.Login = &login
	testState.ServiceLogin = serviceLogin
}
