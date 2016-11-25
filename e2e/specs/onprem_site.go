package specs

import (
	"time"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/constants"
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

		It("should be able to add and remove server", func() {
			ui.EnsureUser(f.Page, siteURL, ctx.Login)

			cluster := framework.Cluster

			By("opening a site page")
			site := uisite.Open(f.Page, domainName)
			site.NavigateToServers()
			siteProvisioner := site.GetSiteServerPage()

			By("executing a command on server")
			agentCommand := siteProvisioner.InitOnPremOperation()
			nodes, err := cluster.Provisioner().NodePool().Allocate(1)
			Expect(err).NotTo(HaveOccurred(), "should allocate a new node")

			framework.RunAgentCommand(agentCommand, nodes[0])

			By("waiting for agent server")
			Eventually(siteProvisioner.GetAgentServers, constants.AgentServerTimeout).Should(
				HaveLen(1),
				"should wait for the agent server")

			By("configuring agent server")
			provisioner := cluster.Provisioner()
			agentServers := siteProvisioner.GetAgentServers()
			for _, s := range agentServers {
				s.SetIPByInfra(provisioner)
			}

			By("starting an expand operation")
			newItem := siteProvisioner.StartOnPremOperation()
			Expect(newItem).NotTo(BeNil(), "new server should appear in the server table")

			time.Sleep(constants.InitializeTimeout)

			By("deleting a server")
			siteProvisioner.DeleteOnPremServer(newItem)
			Expect(cluster.Provisioner().NodePool().Free(nodes)).ShouldNot(
				HaveOccurred(),
				"should dealocate the node after it has been removed")
		})
	})

}
