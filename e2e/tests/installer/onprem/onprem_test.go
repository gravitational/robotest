package onprem

import (
	"time"

	bandwagon "github.com/gravitational/robotest/e2e/asserts/bandwagon"
	validation "github.com/gravitational/robotest/e2e/asserts/installer"
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/ui/installer"
	"github.com/gravitational/robotest/e2e/ui/site"
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OnPrem Installation", func() {
	It("should install an application", func() {
		shouldHandleNewDeploymentScreen()
		shouldHandleRequirementsScreen()
		shouldHandleInProgressScreen()
		shouldHandleBandwagonScreen()
		shouldNavigateToSite()
	})
})

func shouldHandleNewDeploymentScreen() {
	inst := installer.Open(page, framework.TestContext.StartURL)
	By("entering domain name")
	Eventually(inst.IsCreateSiteStep, defaults.FindTimeout).Should(BeTrue())
	inst.CreateOnPremNewSite(framework.TestContext.ClusterName)
}

func shouldHandleRequirementsScreen() {
	inst := installer.OpenWithSite(page, framework.TestContext.ClusterName)
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

func shouldHandleInProgressScreen() {
	validation.WaitForComplete(page, framework.TestContext.ClusterName)
}

func shouldHandleBandwagonScreen() {
	bandwagon.Complete(page, framework.TestContext.ClusterName,
		framework.TestContext.Login.Username,
		framework.TestContext.Login.Password)
}

func shouldNavigateToSite() {
	By("opening a site page")
	site.Open(page, framework.TestContext.ClusterName)
}
