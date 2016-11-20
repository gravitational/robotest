package specs

import (
	"time"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui/installer"
	"github.com/gravitational/robotest/e2e/model/ui/site"
	bandwagon "github.com/gravitational/robotest/e2e/specs/asserts/bandwagon"
	validation "github.com/gravitational/robotest/e2e/specs/asserts/installer"
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
)

func VerifyOnpremInstall(getPage pageFunc, ctx framework.TestContextType, cluster infra.Infra) {

	var (
		page           *agouti.Page
		startURL       = ctx.StartURL
		deploymentName = ctx.ClusterName
		userName       = ctx.Login.Username
		password       = ctx.Login.Password
	)

	Describe("OnPrem Installation", func() {

		BeforeEach(func() {
			page = getPage()
		})

		shouldHandleNewDeploymentScreen := func() {
			inst := installer.Open(page, startURL)
			By("entering domain name")
			Eventually(inst.IsCreateSiteStep, defaults.FindTimeout).Should(BeTrue())
			inst.CreateOnPremNewSite(deploymentName)
		}

		shouldHandleRequirementsScreen := func() {
			inst := installer.OpenWithSite(page, deploymentName)
			Expect(inst.IsRequirementsReviewStep()).To(BeTrue())

			By("selecting a flavor")
			inst.SelectFlavor(1)

			By("veryfing requirements")
			profiles := installer.FindOnPremProfiles(page)
			Expect(len(profiles)).To(Equal(1))

			By("executing the command on servers")
			err := infra.Distribute(profiles[0].Command, cluster.Provisioner().Nodes())
			Expect(err).ShouldNot(HaveOccurred())

			By("waiting for agent report with the servers")
			Eventually(profiles[0].GetServers, 10*time.Minute).Should(HaveLen(1))

			By("veryfing that server has IP")
			server := profiles[0].GetServers()[0]
			ips := server.GetIPs()
			Expect(len(ips) == 2).To(BeTrue())

			By("starting an installation")
			inst.StartInstallation()

			time.Sleep(10 * time.Second)
		}

		shouldHandleInProgressScreen := func() {
			validation.WaitForComplete(page, deploymentName)
		}

		shouldHandleBandwagonScreen := func() {
			bandwagon.Complete(
				page,
				deploymentName,
				userName,
				password)
		}

		shouldNavigateToSite := func() {
			By("opening a site page")
			site.Open(page, deploymentName)
		}

		It("should install an application", func() {
			shouldHandleNewDeploymentScreen()
			shouldHandleRequirementsScreen()
			shouldHandleInProgressScreen()
			shouldHandleBandwagonScreen()
			shouldNavigateToSite()
		})
	})

}
