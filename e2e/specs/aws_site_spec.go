package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	uisite "github.com/gravitational/robotest/e2e/ui/site"
	"github.com/sclevine/agouti"

	. "github.com/onsi/ginkgo"
)

func VerifyAwsSite(getPage pageFunc, ctx framework.TestContextType) {

	var (
		page           *agouti.Page
		deploymentName = ctx.ClusterName
		awsConfig      = ctx.AWS
		profileLabel   = "worker node"
		instanceType   = "m3.large"
	)

	Describe("Aws Site Servers", func() {

		BeforeEach(func() {
			page = getPage()
		})

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

}
