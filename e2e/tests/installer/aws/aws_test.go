package aws

import (
	validation "github.com/gravitational/robotest/e2e/asserts/installer"
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/ui"
	installer "github.com/gravitational/robotest/e2e/ui/installer"
	"github.com/gravitational/robotest/e2e/ui/site"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Installation", func() {
	It("should handle installation", func() {
		shouldHandleNewDeploymentScreen()
		shouldHandleRequirementsScreen()
		shouldHandleInProgressScreen()
		shouldHandleBandwagonScreen()
		shouldNavigateToSite()
	})
})

func shouldNavigateToSite() {
	By("opening a site page")
	site.Open(page, framework.TestContext.ClusterName)
}

func shouldHandleNewDeploymentScreen() {
	inst := installer.Open(page, framework.TestContext.StartURL)
	Eventually(inst.IsCreateSiteStep, defaults.FindTimeout).Should(BeTrue())
	inst.CreateAwsSite(
		framework.TestContext.ClusterName,
		framework.TestContext.AWS,
	)
}

func shouldHandleRequirementsScreen() {
	inst := installer.OpenWithSite(page, framework.TestContext.ClusterName)
	Expect(inst.IsRequirementsReviewStep()).To(BeTrue())

	By("selecting a flavor")
	inst.SelectFlavor(2)

	By("veryfing requirements")
	profiles := installer.FindAwsProfiles(page)
	Expect(len(profiles)).To(Equal(1))

	By("setting instance type")
	profiles[0].SetInstanceType("m3.large")

	By("starting an installation")
	inst.StartInstallation()
}

func shouldHandleInProgressScreen() {
	validation.WaitForComplete(page, framework.TestContext.ClusterName)
}

func shouldHandleBandwagonScreen() {
	By("opening bandwagon page")
	bandwagon := ui.OpenBandwagon(page, framework.TestContext.ClusterName,
		framework.TestContext.Login.Username,
		framework.TestContext.Login.Password)
	By("submitting bandwagon form")
	bandwagon.SubmitForm()
}
