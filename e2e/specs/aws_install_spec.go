package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	installermodel "github.com/gravitational/robotest/e2e/model/ui/installer"
	"github.com/gravitational/robotest/e2e/model/ui/site"
	bandwagon "github.com/gravitational/robotest/e2e/specs/asserts/bandwagon"
	validation "github.com/gravitational/robotest/e2e/specs/asserts/installer"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
)

func VerifyAwsInstall(page *web.Page) {
	Describe("AWS Installation", func() {
		ctx := framework.TestContext
		var domainName string

		BeforeEach(func() {
			domainName = ctx.ClusterName
		})

		shouldNavigateToSite := func() {
			By("opening a site page")
			site.Open(page, domainName)
		}

		shouldHandleNewDeploymentScreen := func() {
			installer := installermodel.Open(page, framework.InstallerURL())

			Eventually(installer.IsCreateSiteStep, defaults.FindTimeout).Should(
				BeTrue(),
				"should navigate to installer screen")

			installer.CreateAwsSite(domainName, *ctx.AWS)
		}

		shouldHandleRequirementsScreen := func() {
			By("entering domain name")
			installer := installermodel.OpenWithSite(page, domainName)
			Expect(installer.IsRequirementsReviewStep()).To(
				BeTrue(),
				"should be on requirement step")

			By("selecting a flavor")
			installer.SelectFlavor(ctx.NumInstallNodes)

			profiles := installermodel.FindAwsProfiles(page)

			Expect(len(profiles)).To(
				Equal(1),
				"should verify required node number")

			profiles[0].SetInstanceType(defaults.InstanceType)

			By("starting an installation")
			installer.StartInstallation()
		}

		shouldHandleInProgressScreen := func() {
			validation.WaitForComplete(page, domainName)
		}

		shouldHandleBandwagonScreen := func() {
			bandwagon.Complete(page, domainName, ctx.Login.Username, ctx.Login.Password)
		}

		It("should handle installation", func() {
			shouldHandleNewDeploymentScreen()
			shouldHandleRequirementsScreen()
			shouldHandleInProgressScreen()
			shouldHandleBandwagonScreen()
			shouldNavigateToSite()
		})
	})
}
