package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	uisite "github.com/gravitational/robotest/e2e/model/ui/site"
	"github.com/gravitational/robotest/lib/defaults"

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

		It("should be able to add and remove server", func() {
			ui.EnsureUser(f.Page, siteURL, ctx.Login)

			By("opening a site page")
			site := uisite.Open(f.Page, domainName)
			site.NavigateToServers()
			siteProvisioner := site.GetSiteServerProvisioner()

			By("executing a command")

			siteProvisioner.InitOnPremOperation()
			/*
				agentCommand := siteProvisioner.InitOnPremOperation()
				node, err := framework.Cluster.Provisioner().Allocate()
				Expect(err).NotTo(HaveOccurred(), "should allocate a new node")
				err = infra.Run(node, agentCommand, os.Stderr)
				Expect(err).NotTo(HaveOccurred(), "should execute command")
			*/

			By("waiting for agent servers")
			Eventually(siteProvisioner.GetAgentServers, defaults.AgentServerTimeout).Should(
				HaveLen(1),
				"should wait for the agent server")

			By("starting an operation")
			newItem := siteProvisioner.StartOnPremOperation()

			Expect(newItem).NotTo(BeNil())

			By("deleting a server")
			siteProvisioner.DeleteOnPremServer(newItem)

			//			cluster.siteProvisioner.AddOnPremServer()

			//		newItem := siteProvisioner.AddAwsServer(awsConfig, profileLabel, instanceType)

		})
	})

}
