package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/ops"
	sitemodel "github.com/gravitational/robotest/e2e/model/ui/site"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func VerifyAWSSiteServers(f *framework.T) {
	framework.RoboDescribe("AWS Site Servers", func() {
		ctx := framework.TestContext
		var domainName string
		var siteURL string

		BeforeEach(func() {
			domainName = ctx.ClusterName
			siteURL = framework.SiteURL()
		})

		It("should be able to add and remove a server", func() {
			ui.EnsureUser(f.Page, siteURL, ctx.Login)

			By("opening a site page")
			site := sitemodel.Open(f.Page, domainName)
			site.NavigateToServers()
			siteProvisioner := site.GetSiteServerPage()

			By("trying to add a new server")
			newItem := siteProvisioner.AddAWSServer(ctx.AWS, ctx.FlavorLabel)

			By("veryfing operation state")
			opsProvider := ops.CreateOpsProvider(f.Page)
			lastExpandOp := opsProvider.GetLastOperationByType(ops.OperationExpandType)
			Expect(lastExpandOp).NotTo(BeNil(), "should retrieve last expand operation")
			Expect(lastExpandOp.State).To(
				BeEquivalentTo(ops.OperationStateCompleted),
				"operation should be in completed state")

			By("removing a server")
			siteProvisioner.DeleteAWSServer(ctx.AWS, newItem)
		})
	})
}
