package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/defaults"
	installermodel "github.com/gravitational/robotest/e2e/model/ui/installer"
	"github.com/gravitational/robotest/e2e/model/ui/site"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func VerifyAWSInstall(f *framework.T) {
	framework.RoboDescribe("AWS Installation", func() {
		ctx := framework.TestContext
		var domainName string
		var siteURL string
		var login = framework.Login{
			Username: defaults.BandwagonEmail,
			Password: defaults.BandwagonPassword,
		}
		var bandwagonConfig = framework.BandwagonConfig{
			Organization: defaults.BandwagonOrganization,
			Username:     defaults.BandwagonUsername,
			Password:     defaults.BandwagonPassword,
			Email:        defaults.BandwagonEmail,
		}

		if ctx.Wizard {
			Skip("this test cannot run in wizard mode")
		}

		BeforeEach(func() {
			domainName = ctx.ClusterName
			siteURL = framework.SiteURL()

			if ctx.Bandwagon.Organization != "" {
				bandwagonConfig.Organization = ctx.Bandwagon.Organization
			}

			if ctx.Bandwagon.Username != "" {
				bandwagonConfig.Username = ctx.Bandwagon.Username
			}

			if ctx.Bandwagon.Password != "" {
				bandwagonConfig.Password = ctx.Bandwagon.Password
				login.Password = ctx.Bandwagon.Password
			}

			if ctx.Bandwagon.Email != "" {
				bandwagonConfig.Email = ctx.Bandwagon.Email
				login.Username = ctx.Bandwagon.Email
			}

		})

		shouldHandleNewDeploymentScreen := func() {
			installer := installermodel.Open(f.Page, framework.InstallerURL())

			By("filling out license text field if required")
			installer.FillOutLicenseIfRequired(ctx.License)

			Eventually(installer.IsCreateSiteStep, defaults.FindTimeout).Should(
				BeTrue(),
				"should navigate to installer screen")

			installer.CreateAWSSite(domainName, ctx.AWS)
		}

		shouldHandleRequirementsScreen := func() {
			By("entering domain name")
			installer := installermodel.OpenWithSite(f.Page, domainName)
			Expect(installer.IsRequirementsReviewStep()).To(
				BeTrue(),
				"should be on requirement step")

			By("selecting a flavor")
			installer.SelectFlavorByLabel(ctx.FlavorLabel)

			profiles := installermodel.FindAWSProfiles(f.Page)

			Expect(len(profiles)).To(
				Equal(1),
				"should verify required node number")

			profiles[0].SetInstanceType(ctx.AWS.InstanceType)

			By("starting an installation")
			installer.StartInstallation()
		}

		shouldHandleInProgressScreen := func() {
			By("waiting until install is completed or failed")
			installer := installermodel.OpenWithSite(f.Page, domainName)
			installer.WaitForComplete()

			By("clicking on continue")
			installer.ProceedToSite()
		}

		shouldHandleBandwagonScreen := func() {
			enableRemoteAccess := true
			By("opening bandwagon page")
			bandwagon := ui.OpenBandwagon(f.Page, domainName, bandwagonConfig)
			By("submitting bandwagon form")
			endpoints := bandwagon.SubmitForm(enableRemoteAccess)
			Expect(len(endpoints)).To(BeNumerically(">", 0))
		}

		shouldNavigateToSite := func() {
			By("opening a site page")
			ui.EnsureUser(f.Page, framework.SiteURL(), login)
			site.Open(f.Page, domainName)
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
