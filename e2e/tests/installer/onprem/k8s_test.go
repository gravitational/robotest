package onprem

import (
	"time"

	"github.com/gravitational/robotest/e2e/ui/installer"
	"github.com/gravitational/robotest/infra"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	defaultTimeout = 20 * time.Second
)

var _ = Describe("OnPrem Installation", func() {
	It("should install an application", func() {
		shouldHandleNewDeploymentScreen()
		shouldHandleRequirementScreen()
	})
})

func shouldHandleNewDeploymentScreen() {
	inst := installer.OpenInstaller(page, startURL)
	By("entering domain name")
	Eventually(inst.IsCreateSiteStep, defaultTimeout).Should(BeTrue())
	inst.CreateOnPremNewSite(domainName)
}

func shouldHandleRequirementScreen() {
	inst := installer.OpenInstallerWithSite(page, domainName)
	Expect(inst.IsRequirementsReviewStep()).To(BeTrue())

	By("selecting a flavor")
	inst.SelectFlavor(2)

	By("veryfing requirements")
	profiles := installer.FindOnPremProfiles(page)
	Expect(len(profiles)).To(Equal(1))

	By("executing the command on servers")
	err := infra.Distribute(profiles[0].Command, provisioner.Nodes())
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
