package specs

import (
	"time"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	installermodel "github.com/gravitational/robotest/e2e/model/ui/installer"
	"github.com/gravitational/robotest/e2e/model/ui/site"
	bandwagon "github.com/gravitational/robotest/e2e/specs/asserts/bandwagon"
	validation "github.com/gravitational/robotest/e2e/specs/asserts/installer"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func VerifyOnpremInstall(f *framework.T) {
	var _ = framework.RoboDescribe("Onprem Installation", func() {
		ctx := framework.TestContext
		var domainName string

		BeforeEach(func() {
			domainName = ctx.ClusterName
		})

		shouldHandleNewDeploymentScreen := func() {
			installer := installermodel.Open(f.Page, framework.InstallerURL())
			By("entering domain name")
			Eventually(installer.IsCreateSiteStep, defaults.FindTimeout).Should(BeTrue())
			installer.CreateOnPremNewSite(domainName)
		}

		shouldHandleRequirementsScreen := func() {
			installer := installermodel.OpenWithSite(f.Page, domainName)
			Expect(installer.IsRequirementsReviewStep()).To(BeTrue())

			By("selecting a flavor")
			installer.SelectFlavor(ctx.NumInstallNodes)

			By("veryfing requirements")
			profiles := installermodel.FindOnPremProfiles(f.Page)
			Expect(len(profiles)).To(Equal(1))

			By("executing the command on servers")
			framework.RunAgentCommand(profiles[0].Command)

			By("waiting for agent report with the servers")
			Eventually(profiles[0].GetAgentServers, defaults.AgentServerTimeout).Should(HaveLen(1))

			By("veryfing that server has IP")
			server := profiles[0].GetAgentServers()[0]
			ips := server.GetIPs()
			Expect(len(ips)).To(BeNumerically(">", 0))
			// FIXME: make sure there're ctx.NumInstallNodes agent entries

			By("starting an installation")
			installer.StartInstallation()

			time.Sleep(defaults.ShortTimeout)
		}

		shouldHandleInProgressScreen := func() {
			validation.WaitForComplete(f.Page, domainName)
		}

		shouldHandleBandwagonScreen := func() {
			bandwagon.Complete(
				f.Page,
				domainName,
				ctx.Login.Username,
				ctx.Login.Password)
		}

		shouldNavigateToSite := func() {
			By("opening a site page")
			site.Open(f.Page, domainName)
		}

		It("should install an application", func() {
			ui.EnsureUser(f.Page, framework.InstallerURL(), ctx.Login)
			shouldHandleNewDeploymentScreen()
			shouldHandleRequirementsScreen()
			shouldHandleInProgressScreen()
			shouldHandleBandwagonScreen()
			shouldNavigateToSite()
		})
	})
}
