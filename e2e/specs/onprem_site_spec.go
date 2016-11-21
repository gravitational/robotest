package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	uisite "github.com/gravitational/robotest/e2e/model/ui/site"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func VerifyOnpremSite(f *framework.T) {

	var _ = framework.RoboDescribe("Onprem Site Servers", func() {

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
			site := uisite.Open(f.Page, domainName)
			site.NavigateToServers()
			siteProvisioner := site.GetSiteServerProvisioner()

			siteProvisioner.InitOnPremOperation()

			By("trying to add a new server")
			// infra.Distribute(command, cluster.Provisioner().Create())

			newItem := siteProvisioner.StartOnPremOperation()

			Expect(newItem).NotTo(BeNil())

			//			cluster.siteProvisioner.AddOnPremServer()

			//		newItem := siteProvisioner.AddAwsServer(awsConfig, profileLabel, instanceType)

		})
	})

}
