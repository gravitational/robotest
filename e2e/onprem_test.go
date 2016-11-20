package e2e

import (
	bandwagon "github.com/gravitational/robotest/e2e/asserts/bandwagon"
	validation "github.com/gravitational/robotest/e2e/asserts/installer"
	"github.com/gravitational/robotest/e2e/framework"
	installermodel "github.com/gravitational/robotest/e2e/model/ui/installer"
	sitemodel "github.com/gravitational/robotest/e2e/model/ui/site"
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
)

var _ = framework.RoboDescribe("Onprem installation", func() {
	f := framework.New()
	var page *web.Page

	BeforeEach(func() {
		if page == nil {
			page = f.InstallerPage(framework.WithGoogle)
		}
	})

	It("should create a new deployment", func() {
		installerURL, err := framework.InstallerURL()
		Expect(err).NotTo(HaveOccurred())
		installer := installermodel.Open(page, installerURL)
		By("entering domain name")
		Eventually(installer.IsCreateSiteStep, defaults.FindTimeout).Should(BeTrue())
		installer.CreateOnPremNewSite(framework.TestContext.ClusterName)
	})

	It("should fill out requirements screen", func() {
		installer := installermodel.OpenWithSite(page, framework.TestContext.ClusterName)
		Expect(installer.IsRequirementsReviewStep()).To(BeTrue())

		By("selecting a flavor")
		installer.SelectFlavor(framework.TestContext.NumInstallNodes)

		By("veryfing requirements")
		profiles := installermodel.FindOnPremProfiles(page)
		Expect(len(profiles)).To(Equal(1))

		By("executing the command on servers")
		err := infra.Distribute(profiles[0].Command, framework.Cluster.Provisioner().Nodes())
		Expect(err).NotTo(HaveOccurred())

		By("waiting for agent report with the servers")
		Eventually(profiles[0].GetServers, defaults.AgentTimeout).Should(HaveLen(1))

		By("veryfing that server has IP")
		server := profiles[0].GetServers()[0]
		ips := server.GetIPs()
		Expect(len(ips) == framework.TestContext.NumInstallNodes).To(BeTrue())

		By("starting an installation")
		installer.StartInstallation()
	})

	It("should wait for completion", func() {
		validation.WaitForComplete(page, framework.TestContext.ClusterName)
	})

	It("should wait for completion", func() {
		bandwagon.Complete(page, framework.TestContext.ClusterName,
			framework.TestContext.Login.Username,
			framework.TestContext.Login.Password)
	})

	It("should wait for completion", func() {
		sitemodel.Open(page, framework.TestContext.ClusterName)
	})
})
