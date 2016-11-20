package e2e

import (
	"github.com/gravitational/robotest/e2e/asserts/bandwagon"
	validation "github.com/gravitational/robotest/e2e/asserts/installer"
	"github.com/gravitational/robotest/e2e/framework"
	installermodel "github.com/gravitational/robotest/e2e/model/ui/installer"
	sitemodel "github.com/gravitational/robotest/e2e/model/ui/site"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
)

var _ = framework.RoboDescribe("AWS installation", func() {
	f := framework.New()
	ctx := framework.TestContext
	var page *web.Page

	BeforeEach(func() {
		if page == nil {
			page = f.InstallerPage(framework.WithGoogle)
		}
	})

	It("should create a new deployment", func() {
		url, err := framework.InstallerURL()
		Expect(err).NotTo(HaveOccurred())

		installer := installermodel.Open(page, url)
		config := *ctx.AWS
		Eventually(installer.IsCreateSiteStep, defaults.FindTimeout).Should(BeTrue())
		installer.CreateAwsSite(ctx.ClusterName, config)
	})

	It("should fill out requirements screen", func() {
		installer := installermodel.OpenWithSite(page, ctx.ClusterName)
		Expect(installer.IsRequirementsReviewStep()).To(BeTrue())

		By("selecting a flavor")
		installer.SelectFlavor(ctx.NumInstallNodes)

		By("veryfing requirements")
		profiles := installermodel.FindAwsProfiles(page)
		Expect(len(profiles)).To(Equal(1))

		By("setting instance type")
		profiles[0].SetInstanceType(defaults.InstanceType)

		By("starting an installation")
		installer.StartInstallation()
	})

	It("should wait for completion", func() {
		validation.WaitForComplete(page, ctx.ClusterName)
	})

	It("should fill out bandwagon screen", func() {
		bandwagon.Complete(page, ctx.ClusterName, ctx.Login.Username, ctx.Login.Password)
	})

	It("should navigate to installed site", func() {
		sitemodel.Open(page, ctx.ClusterName)
	})
})
