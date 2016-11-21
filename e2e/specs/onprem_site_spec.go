package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	uisite "github.com/gravitational/robotest/e2e/ui/site"
	"github.com/gravitational/robotest/infra"
	"github.com/sclevine/agouti"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func VerifyOnpremSite(getPage pageFunc, ctx framework.TestContextType, getCluster clusterFunc) {

	var (
		cluster        infra.Infra
		page           *agouti.Page
		deploymentName = ctx.ClusterName
	)

	Describe("Onprem Site Servers", func() {

		BeforeEach(func() {
			page = getPage()
			cluster = getCluster()
		})

		It("should be able to add and remove a server", func() {
			By("opening a site page")
			site := uisite.Open(page, deploymentName)
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
