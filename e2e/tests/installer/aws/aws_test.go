package aws

import (
	bandwagon "github.com/gravitational/robotest/e2e/asserts/bandwagon"
	validation "github.com/gravitational/robotest/e2e/asserts/installer"
	"github.com/gravitational/robotest/e2e/framework"
	installer "github.com/gravitational/robotest/e2e/ui/installer"
	"github.com/gravitational/robotest/e2e/ui/site"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	startURL           = ""
	deploymentName     = ""
	userName           = ""
	password           = ""
	awsConfig          framework.AWSConfig
	flavorIndex        = 2
	serverInstanceType = "m3.large"
)

var _ = Describe("Installation", func() {

	BeforeEach(func() {
		deploymentName = framework.TestContext.ClusterName
		startURL = framework.TestContext.StartURL
		userName = framework.TestContext.Login.Username
		password = framework.TestContext.Login.Password
		awsConfig = framework.TestContext.AWS
	})

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
	site.Open(page, deploymentName)
}

func shouldHandleNewDeploymentScreen() {
	inst := installer.Open(page, startURL)
	Eventually(inst.IsCreateSiteStep, defaults.FindTimeout).Should(BeTrue())
	inst.CreateAwsSite(deploymentName, awsConfig)
}

func shouldHandleRequirementsScreen() {
	inst := installer.OpenWithSite(page, deploymentName)
	Expect(inst.IsRequirementsReviewStep()).To(BeTrue())

	By("selecting a flavor")
	inst.SelectFlavor(flavorIndex)

	By("veryfing requirements")
	profiles := installer.FindAwsProfiles(page)
	Expect(len(profiles)).To(Equal(1))

	By("setting instance type")
	profiles[0].SetInstanceType(serverInstanceType)

	By("starting an installation")
	inst.StartInstallation()
}

func shouldHandleInProgressScreen() {
	validation.WaitForComplete(page, deploymentName)
}

func shouldHandleBandwagonScreen() {
	bandwagon.Complete(page, deploymentName, userName, password)
}
