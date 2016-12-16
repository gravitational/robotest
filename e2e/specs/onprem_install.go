package specs

import (
	"fmt"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/defaults"
	installermodel "github.com/gravitational/robotest/e2e/model/ui/installer"
	"github.com/gravitational/robotest/e2e/model/ui/site"
	bandwagon "github.com/gravitational/robotest/e2e/specs/asserts/bandwagon"
	validation "github.com/gravitational/robotest/e2e/specs/asserts/installer"
	"github.com/gravitational/robotest/infra"

	log "github.com/Sirupsen/logrus"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func VerifyOnpremInstall(f *framework.T) {
	var _ = framework.RoboDescribe("Onprem Installation", func() {
		ctx := framework.TestContext
		var domainName string
		var login = framework.Login{
			Username: defaults.BandwagonUsername,
			Password: defaults.BandwagonPassword,
		}

		var bandwagonLogin = framework.BandwagonConfig{
			Organization: defaults.BandwagonOrganization,
			Username:     defaults.BandwagonUsername,
			Password:     defaults.BandwagonPassword,
			Email:        defaults.BandwagonEmail,
		}

		if ctx.Bandwagon.Organization != "" {
			bandwagonLogin.Organization = ctx.Bandwagon.Organization
		}
		if ctx.Bandwagon.Username != "" {
			bandwagonLogin.Username = ctx.Bandwagon.Username
		}
		if ctx.Bandwagon.Password != "" {
			bandwagonLogin.Password = ctx.Bandwagon.Password
		}
		if ctx.Bandwagon.Email != "" {
			bandwagonLogin.Email = ctx.Bandwagon.Email
		}

		// installNode is the node used to install application on
		var installNode infra.Node

		BeforeEach(func() {
			domainName = ctx.ClusterName
		})

		shouldHandleNewDeploymentScreen := func() {
			installer := installermodel.Open(f.Page, framework.InstallerURL())
			By("filling out license text field if required")
			installer.FillOutLicenseIfRequired(ctx.License)
			By("entering domain name")
			Eventually(installer.IsCreateSiteStep, defaults.FindTimeout).Should(BeTrue())
			installer.CreateOnPremNewSite(domainName)
		}

		shouldHandleRequirementsScreen := func() {
			installer := installermodel.OpenWithSite(f.Page, domainName)
			Expect(installer.IsRequirementsReviewStep()).To(BeTrue())

			By("selecting a flavor")
			numInstallNodes := installer.SelectFlavorByLabel(ctx.FlavorLabel)

			provisioner := framework.Cluster.Provisioner()
			Expect(provisioner).NotTo(BeNil(), "expected valid provisioner for onprem installation")
			log.Infof("allocating %v nodes", numInstallNodes)

			var err error
			var allocated []infra.Node
			allocated, err = provisioner.NodePool().Allocate(numInstallNodes)
			Expect(err).NotTo(HaveOccurred(), "expected to allocate node(s)")
			Expect(allocated).To(HaveLen(numInstallNodes),
				fmt.Sprintf("expected to allocated %v nodes", numInstallNodes))

			// Use the first allocated node to access the local site
			installNode = allocated[0]

			By("veryfing requirements")
			profiles := installermodel.FindOnPremProfiles(f.Page)
			Expect(len(profiles)).To(Equal(1))

			By("executing the command on servers")
			framework.RunAgentCommand(profiles[0].Command)

			By("waiting for agent report with the servers")
			Eventually(profiles[0].GetAgentServers, defaults.AgentServerTimeout).Should(
				HaveLen(numInstallNodes))

			By("configuring the servers with IPs")
			agentServers := profiles[0].GetAgentServers()

			for _, s := range agentServers {
				s.SetIPByInfra(provisioner)
			}

			By("starting an installation")
			installer.StartInstallation()
		}

		shouldHandleInProgressScreen := func() {
			validation.WaitForComplete(f.Page, domainName)
		}

		shouldHandleBandwagonScreen := func() {
			enableRemoteAccess := ctx.ForceRemoteAccess || !ctx.Wizard
			// useLocalEndpoint := ctx.ForceLocalEndpoint || ctx.Wizard
			endpoints := bandwagon.Complete(
				f.Page,
				domainName,
				bandwagonLogin,
				enableRemoteAccess)
			By("using local application endpoint")
			serviceLogin := &framework.ServiceLogin{Username: login.Username, Password: login.Password}
			siteEntryURL := endpoints[0]
			// TODO: for terraform, use public install node address
			// terraform nodes are provisioned only with a single private network interface
			if ctx.Provisioner == "terraform" {
				siteEntryURL = fmt.Sprintf("https://%v:%v", installNode.Addr(), defaults.GravityHTTPPort)
			}
			framework.UpdateSiteEntry(siteEntryURL, login, serviceLogin)
		}

		shouldNavigateToSite := func() {
			By("opening a site page")
			ui.EnsureUser(f.Page, framework.SiteURL(), login)
			site.Open(f.Page, domainName)
		}

		It("should install an application", func() {
			ui.EnsureUser(f.Page, framework.InstallerURL(), ctx.Login)
			shouldHandleNewDeploymentScreen()
			shouldHandleRequirementsScreen()
			shouldHandleInProgressScreen()
			shouldHandleBandwagonScreen()
			shouldNavigateToSite()
		})
	})
}
