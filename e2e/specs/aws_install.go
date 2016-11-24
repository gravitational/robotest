package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/constants"
	installermodel "github.com/gravitational/robotest/e2e/model/ui/installer"
	"github.com/gravitational/robotest/e2e/model/ui/site"
	bandwagon "github.com/gravitational/robotest/e2e/specs/asserts/bandwagon"
	validation "github.com/gravitational/robotest/e2e/specs/asserts/installer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func VerifyAwsInstall(f *framework.T) {
	Describe("AWS Installation", func() {
		ctx := framework.TestContext
		var domainName string
		var siteURL string

		BeforeEach(func() {
			domainName = ctx.ClusterName
			siteURL = framework.SiteURL()
		})

		shouldNavigateToSite := func() {
			By("opening a site page")
			site.Open(f.Page, domainName)
		}

		shouldHandleNewDeploymentScreen := func() {
			installer := installermodel.Open(f.Page, framework.InstallerURL())

			Eventually(installer.IsCreateSiteStep, constants.FindTimeout).Should(
				BeTrue(),
				"should navigate to installer screen")

			installer.CreateAwsSite(domainName, ctx.AWS)
		}

		shouldHandleRequirementsScreen := func() {
			By("entering domain name")
			installer := installermodel.OpenWithSite(f.Page, domainName)
			Expect(installer.IsRequirementsReviewStep()).To(
				BeTrue(),
				"should be on requirement step")

			By("selecting a flavor")
			installer.SelectFlavor(ctx.NumInstallNodes)

			profiles := installermodel.FindAwsProfiles(f.Page)

			Expect(len(profiles)).To(
				Equal(1),
				"should verify required node number")

			profiles[0].SetInstanceType(awsInstanceType)

			By("starting an installation")
			installer.StartInstallation()
		}

		shouldHandleInProgressScreen := func() {
			validation.WaitForComplete(f.Page, domainName)
		}

		shouldHandleBandwagonScreen := func() {
			bandwagon.Complete(f.Page, domainName, ctx.Login.Username, ctx.Login.Password)
		}

		It("should handle installation", func() {
			ui.EnsureUser(f.Page, framework.InstallerURL(), ctx.Login)
			shouldHandleNewDeploymentScreen()
			shouldHandleRequirementsScreen()
			shouldHandleInProgressScreen()
			shouldHandleBandwagonScreen()
			shouldNavigateToSite()
		})
	})
}
