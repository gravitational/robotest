package specs

import (
	"fmt"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/defaults"
	installermodel "github.com/gravitational/robotest/e2e/model/ui/installer"
	"github.com/gravitational/robotest/e2e/model/ui/site"

	log "github.com/Sirupsen/logrus"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func VerifyOnpremInstall(f *framework.T) {
	var _ = framework.RoboDescribe("Onprem Installation", func() {
		ctx := framework.TestContext
		var domainName string
		// docker default device will be loopback
		var dockerDevice string
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

		BeforeEach(func() {
			domainName = ctx.ClusterName

			if ctx.Onprem.DockerDevice != "" {
				dockerDevice = ctx.Onprem.DockerDevice
			}

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
			By("entering domain name")
			Eventually(installer.IsCreateSiteStep, defaults.FindTimeout).Should(BeTrue())
			installer.CreateOnPremNewSite(domainName)
		}

		shouldHandleRequirementsScreen := func() {
			installer := installermodel.OpenWithSite(f.Page, domainName)
			Expect(installer.IsRequirementsReviewStep()).To(BeTrue())

			By("selecting a flavor")
			numInstallNodes := installer.SelectFlavorByLabel(ctx.FlavorLabel)
			profiles := installermodel.FindOnPremProfiles(f.Page)
			Expect(len(profiles)).NotTo(Equal(0))

			By("allocating nodes")
			log.Infof("allocating %v nodes", numInstallNodes)
			provisioner := framework.Cluster.Provisioner()
			Expect(provisioner).NotTo(BeNil(), "expected valid provisioner for onprem installation")
			allocatedNodes, err := provisioner.NodePool().Allocate(numInstallNodes)
			Expect(err).NotTo(HaveOccurred(), "expected to allocate node(s)")
			Expect(allocatedNodes).To(HaveLen(numInstallNodes),
				fmt.Sprintf("expected to allocated %v nodes", numInstallNodes))

			By("executing the command on servers")
			index := 0
			for _, p := range profiles {
				nodesForProfile := allocatedNodes[index:p.Count]
				framework.RunAgentCommand(p.Command, nodesForProfile...)
				index = index + p.Count
			}

			By("waiting for agent report with the servers")
			for _, p := range profiles {
				Eventually(p.GetAgentServers, defaults.AgentServerTimeout).Should(
					HaveLen(p.Count))
			}

			By("configuring the servers with IPs")
			for _, p := range profiles {
				agentServers := p.GetAgentServers()
				for _, s := range agentServers {
					s.SetIPByInfra(provisioner)
					if s.NeedsDockerDevice() {
						s.SetDockerDevice(dockerDevice)
					}
				}
			}

			By("starting an installation")
			installer.StartInstallation()
		}

		shouldHandleInProgressScreen := func() {
			By("waiting until install is completed")
			installer := installermodel.OpenWithSite(f.Page, domainName)
			installer.WaitForComplete()
		}

		shouldHandleBandwagonScreen := func() {
			installer := installermodel.OpenWithSite(f.Page, domainName)
			if ui.NeedsBandwagon(f.Page, domainName) == false {
				return
			}

			By("clicking on continue")
			installer.ProceedToSite()

			// Use the first allocated node to access the local site
			allocatedNodes := framework.Cluster.Provisioner().NodePool().AllocatedNodes()
			installNode := allocatedNodes[0]

			enableRemoteAccess := ctx.ForceRemoteAccess || !ctx.Wizard
			bandwagon := ui.OpenBandwagon(f.Page, domainName, bandwagonConfig)
			By("submitting bandwagon form")
			endpoints := bandwagon.SubmitForm(enableRemoteAccess)
			Expect(len(endpoints)).To(BeNumerically(">", 0))

			By("using local application endpoint")
			serviceLogin := &framework.ServiceLogin{Username: login.Username, Password: login.Password}
			siteEntryURL := endpoints[0]
			// For terraform, use public install node address
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
