package aws

import (
	"time"

	installAsserts "github.com/gravitational/robotest/e2e/asserts/installer"
	"github.com/gravitational/robotest/e2e/ui"
	uiInstaller "github.com/gravitational/robotest/e2e/ui/installer"
	uiSite "github.com/gravitational/robotest/e2e/ui/site"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	defaultTimeout = 20 * time.Second
)

var _ = Describe("Installation", func() {

	It("should handle installation", func() {
		shouldHandleNewDeploymentScreen()
		shouldHandleRequirementScreen()
		shouldHandleInProgressScreen()
		shouldHandleBandWagonScreen()
		shouldNavigateToSite()
	})

})

func shouldNavigateToSite() {
	By("opening a site page")
	uiSite.OpenSite(page, deploymentName)
}

func shouldHandleNewDeploymentScreen() {
	inst := uiInstaller.OpenInstaller(page, startURL)
	Eventually(inst.IsCreateSiteStep, defaultTimeout).Should(BeTrue())
	inst.CreateAwsSite(
		deploymentName,
		awsAccessKey,
		awsSecretKey,
		awsRegion,
		awsKeyPair,
		awsVpc,
	)
}

func shouldHandleRequirementScreen() {
	inst := uiInstaller.OpenInstallerWithSite(page, deploymentName)
	Expect(inst.IsRequirementsReviewStep()).To(BeTrue())

	By("selecting a flavor")
	inst.SelectFlavor(2)

	By("veryfing requirements")
	profiles := uiInstaller.FindAwsProfiles(page)
	Expect(len(profiles)).To(Equal(1))

	By("setting instance type")
	profiles[0].SetInstanceType("m3.large")

	By("starting an installation")
	inst.StartInstallation()
}

func shouldHandleInProgressScreen() {
	installAsserts.WaitForComplete(page, deploymentName)
}

func shouldHandleBandWagonScreen() {
	By("opening bandwagon page")
	bandwagon := ui.OpenBandWagon(page, deploymentName, userName, password)
	By("submitting bandwagon form")
	bandwagon.SubmitForm()
}
