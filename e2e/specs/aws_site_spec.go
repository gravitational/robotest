package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	sitemodel "github.com/gravitational/robotest/e2e/model/ui/site"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
)

func VerifyAwsSite(f *framework.T) {
	Describe("AWS Site Servers", func() {
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
			siteProvisioner := site.GetSiteServerProvisioner()

			By("trying to add a new server")
			newItem := siteProvisioner.AddAwsServer(ctx.AWS, defaults.ProfileLabel, defaults.InstanceType)

			By("trying remove a server")
			siteProvisioner.DeleteAwsServer(ctx.AWS, newItem)
		})
	})
}
