package site

import (
	uisite "github.com/gravitational/robotest/e2e/ui/site"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Aws Site Servers", func() {
	It("should be able to add and remove a server", func() {
		By("opening a site page")
		site := uisite.Open(page, deploymentName)
		site.NavigateToServers()
		siteProvisioner := site.GetSiteServerProvisioner()

		By("trying to add a new server")
		newItem := siteProvisioner.AddAwsServer(awsConfig, profileLabel, instanceType)

		By("trying remove a server")
		siteProvisioner.DeleteAwsServer(awsConfig, newItem)

	})
})
