package e2e

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gravitational/robotest/e2e/framework"

	"github.com/gravitational/robotest/e2e/model/ui"
	uidefaults "github.com/gravitational/robotest/e2e/model/ui/defaults"

	installermodel "github.com/gravitational/robotest/e2e/model/ui/installer"
	sitemodel "github.com/gravitational/robotest/e2e/model/ui/site"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = framework.RoboDescribe("Onprem Integration Test", func() {
	f := framework.New()
	ctx := framework.TestContext
	var domainName string

	BeforeEach(func() {
		domainName = ctx.ClusterName
	})

	framework.RoboDescribe("Provisioning a new cluster [provisioner:onprem][install]", func() {
		It("should provision a new cluster", func() {
			By("navigating to installer step")
			ui.EnsureUser(f.Page, framework.InstallerURL(), ctx.Login)
			installer := installermodel.Open(f.Page, framework.InstallerURL())

			By("filling out license text field if required")
			installer.ProcessLicenseStepIfRequired(ctx.License)

			By("selecting a provisioner")
			installer.CreateOnPremNewSite(domainName)

			By("selecting a flavor and allocating the nodes")
			installer.SelectFlavorByLabel(ctx.FlavorLabel)
			profiles := installermodel.FindOnPremProfiles(f.Page)
			Expect(len(profiles)).NotTo(Equal(0))
			installer.PrepareOnPremNodes(ctx.Onprem.DockerDevice)

			By("starting an installation")
			installer.StartInstallation()

			By("waiting until install is completed")
			installer.WaitForComplete()

			By("checking for bandwagon step")
			if installer.NeedsBandwagon() == false {
				sitemodel.Open(f.Page, domainName)
				return
			}

			By("navigating to bandwagon step")
			installer.ProceedToSite()
			bandwagon := ui.OpenBandwagon(f.Page, domainName)
			By("submitting bandwagon form")
			enableRemoteAccess := ctx.ForceRemoteAccess || !ctx.Wizard
			ctx.Bandwagon.RemoteAccess = enableRemoteAccess
			bandwagon.SubmitForm(ctx.Bandwagon)

			By("navigating to a site and reading endpoints")
			site := sitemodel.Open(f.Page, domainName)
			endpoints := site.GetEndpoints()
			endpoints = filterGravityEndpoints(endpoints)
			Expect(len(endpoints)).To(BeNumerically(">", 0), "expected at least one application endpoint")

			By("using local application endpoint")
			var login = framework.Login{
				Username: uidefaults.BandwagonEmail,
				Password: uidefaults.BandwagonPassword,
			}

			// Use the first allocated node to access the local site
			allocatedNodes := framework.Cluster.Provisioner().NodePool().AllocatedNodes()
			siteEntryURL := endpoints[0]
			// For terraform, use public install node address
			// terraform nodes are provisioned only with a single private network interface
			if ctx.Provisioner == "terraform" {
				installNode := allocatedNodes[0]
				siteEntryURL = fmt.Sprintf("https://%v:%v", installNode.Addr(), uidefaults.GravityHTTPPort)
			}
			serviceLogin := &framework.ServiceLogin{Username: login.Username, Password: login.Password}
			By("login in with bandwagon user credentials")
			framework.UpdateSiteEntry(siteEntryURL, login, serviceLogin)
			ui.EnsureUser(f.Page, framework.SiteURL(), ctx.Login)
			sitemodel.Open(f.Page, domainName)
		})
	})

	framework.RoboDescribe("Site expand|shrink operation [provisioner:onprem][expand][shrink]", func() {
		var siteServer = sitemodel.SiteServer{}
		var site = sitemodel.Site{}

		BeforeEach(func() {
			ui.EnsureUser(f.Page, framework.SiteURL(), ctx.Login)
			site = sitemodel.Open(f.Page, domainName)
			site.NavigateToServers()
		})

		It("should add a new server", func() {
			siteServerPage := site.GetSiteServerPage()
			siteServer = siteServerPage.AddOnPremServer()
		})

		It("should remove a new server", func() {
			siteServerPage := site.GetSiteServerPage()
			siteServerPage.DeleteServer(siteServer)
		})
	})

	framework.RoboDescribe("Site update operation [provisioner:onprem][update]", func() {
		It("should update site to the latest version", func() {
			siteURL := framework.SiteURL()
			By("uploading new application into site")
			if ctx.Onprem.InstallerURL == "" {
				// Upload a new version to Ops Center
				// TODO: remove the fake version at cleanup/teardown
				framework.FakeUpdateApplication()
			} else {
				framework.UpdateApplicationWithInstaller()
			}

			By("trying to update the site to the latest application version")
			ui.EnsureUser(f.Page, siteURL, ctx.Login)
			site := sitemodel.Open(f.Page, domainName)
			site.UpdateToLatestVersion()
		})
	})
})

func filterGravityEndpoints(endpoints []string) []string {
	var siteEndpoints []string
	for _, v := range endpoints {
		if strings.Contains(v, strconv.Itoa(uidefaults.GravityHTTPPort)) {
			siteEndpoints = append(siteEndpoints, v)
		}
	}

	return siteEndpoints
}
