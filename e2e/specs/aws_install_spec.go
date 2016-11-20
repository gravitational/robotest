package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	installer "github.com/gravitational/robotest/e2e/model/ui/installer"
	"github.com/gravitational/robotest/e2e/model/ui/site"
	bandwagon "github.com/gravitational/robotest/e2e/specs/asserts/bandwagon"
	validation "github.com/gravitational/robotest/e2e/specs/asserts/installer"
	"github.com/gravitational/robotest/lib/defaults"
	"github.com/sclevine/agouti"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func VerifyAwsInstall(getPage pageFunc, ctx framework.TestContextType) {

	var (
		page               *agouti.Page
		deploymentName     = ctx.ClusterName
		appToInstallURL    = ctx.StartURL
		userName           = ctx.Login.Username
		password           = ctx.Login.Password
		awsConfig          = ctx.AWS
		flavorIndex        = 2
		serverInstanceType = "m3.large"
	)

	Describe("AWS Installation", func() {

		BeforeEach(func() {
			page = getPage()
		})

		shouldNavigateToSite := func() {
			By("opening a site page")
			site.Open(page, deploymentName)
		}

		shouldHandleNewDeploymentScreen := func() {
			inst := installer.Open(page, appToInstallURL)

			Eventually(inst.IsCreateSiteStep, defaults.FindTimeout).Should(
				BeTrue(),
				"Should navigate to installer screen")

			inst.CreateAwsSite(deploymentName, awsConfig)
		}

		shouldHandleRequirementsScreen := func() {
			By("entering domain name")
			inst := installer.OpenWithSite(page, deploymentName)
			Expect(inst.IsRequirementsReviewStep()).To(
				BeTrue(),
				"Should be on requirement step")

			By("selecting a flavor")
			inst.SelectFlavor(flavorIndex)

			profiles := installer.FindAwsProfiles(page)

			Expect(len(profiles)).To(
				Equal(1),
				"Should verify required node number")

			profiles[0].SetInstanceType(serverInstanceType)

			By("starting an installation")
			inst.StartInstallation()
		}

		shouldHandleInProgressScreen := func() {
			validation.WaitForComplete(page, deploymentName)
		}

		shouldHandleBandwagonScreen := func() {
			bandwagon.Complete(page, deploymentName, userName, password)
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
