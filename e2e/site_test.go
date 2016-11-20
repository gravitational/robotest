package e2e

import (
	"github.com/gravitational/robotest/e2e/framework"
	sitemodel "github.com/gravitational/robotest/e2e/model/ui/site"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Aws Site Servers", func() {
	f := framework.New()
	ctx := framework.TestContext

	It("should be able to add and remove a server", func() {
		By("opening a site page")
		site := sitemodel.Open(f.Page, ctx.ClusterName)
		site.NavigateToServers()
		siteProvisioner := site.GetSiteServerProvisioner()

		By("adding a new server")
		newItem := siteProvisioner.AddAwsServer(*ctx.AWS, defaults.ProfileLabel, defaults.InstanceType)

		By("removing a server")
		siteProvisioner.DeleteAwsServer(*ctx.AWS, newItem)
	})
})
