package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	sitemodel "github.com/gravitational/robotest/e2e/model/ui/site"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
	web "github.com/sclevine/agouti"
)

func VerifyAwsSite(page *web.Page) {
	Describe("AWS Site Servers", func() {
		ctx := framework.TestContext
		var domainName string

		BeforeEach(func() {
			domainName = ctx.ClusterName
		})

		It("should be able to add and remove a server", func() {
			By("opening a site page")
			site := sitemodel.Open(page, domainName)
			site.NavigateToServers()
			siteProvisioner := site.GetSiteServerProvisioner()

			By("trying to add a new server")
			newItem := siteProvisioner.AddAwsServer(*ctx.AWS, defaults.ProfileLabel, defaults.InstanceType)

			By("trying remove a server")
			siteProvisioner.DeleteAwsServer(*ctx.AWS, newItem)
		})
	})
}
