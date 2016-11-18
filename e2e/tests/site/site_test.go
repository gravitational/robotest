package site

import (
	ui "github.com/gravitational/robotest/e2e/ui/site"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Aws Site Servers", func() {
	It("should be able to add and remove a server", func() {
		By("opening a site page")
		site := ui.OpenSite(page, deploymentName)
		site.NavigateToServers()
		siteProvisioner := site.GetSiteServerProvisioner()

		By("trying to add a new server")
		newItem := siteProvisioner.AddAwsServer(awsAccessKey, awsSecretKey, profileLabel, instanceType)

		By("trying remove a server")
		siteProvisioner.DeleteAwsServer(awsAccessKey, awsSecretKey, newItem)

	})
})
