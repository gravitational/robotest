package e2e

import (
	"github.com/gravitational/robotest/e2e/framework"

	"github.com/gravitational/robotest/e2e/model/ui"
	installermodel "github.com/gravitational/robotest/e2e/model/ui/installer"
	opscentermodel "github.com/gravitational/robotest/e2e/model/ui/opscenter"
	sitemodel "github.com/gravitational/robotest/e2e/model/ui/site"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = framework.RoboDescribe("AWS Integration Test", func() {
	f := framework.New()
	ctx := framework.TestContext

	var domainName string

	BeforeEach(func() {
		domainName = ctx.ClusterName
	})

	framework.RoboDescribe("Provisioning a new cluster [provisioner:aws][install]", func() {
		It("should provision a new cluster", func() {
			ui.EnsureUser(f.Page, framework.InstallerURL(), ctx.Login)
			installer := installermodel.Open(f.Page, framework.InstallerURL())

			By("filling out license text field if required")
			installer.ProcessLicenseStepIfRequired(ctx.License)
			installer.CreateAWSSite(domainName)

			By("selecting a flavor")
			installer.SelectFlavorByLabel(ctx.FlavorLabel)
			profiles := installer.GetAWSProfiles()
			Expect(len(profiles)).To(Equal(1), "should verify required node number")

			By("setting up AWS instance types")
			profiles[0].SetInstanceType(ctx.AWS.InstanceType)

			By("starting an installation")
			installer.StartInstallation()

			By("waiting until install is completed or failed")
			installer.WaitForComplete()

			if installer.NeedsBandwagon() == true {
				By("navigating to bandwagon step")
				bandwagon := ui.OpenBandwagon(f.Page, domainName)
				By("submitting bandwagon form")
				enableRemoteAccess := ctx.ForceRemoteAccess || !ctx.Wizard
				ctx.Bandwagon.RemoteAccess = enableRemoteAccess
				bandwagon.SubmitForm(ctx.Bandwagon)

				By("navigating to a site and reading endpoints")
				site := sitemodel.Open(f.Page, domainName)
				endpoints := site.GetEndpoints()
				Expect(len(endpoints)).To(BeNumerically(">", 0), "expected at least one application endpoint")
			} else {
				By("clicking on continue")
				installer.ProceedToSite()
			}
		})
	})

	framework.RoboDescribe("Site expand and shrink operations [provisioner:aws][expand][shrink]", func() {
		var siteServer = sitemodel.SiteServer{}
		var site = sitemodel.Site{}

		BeforeEach(func() {
			ui.EnsureUser(f.Page, framework.SiteURL(), ctx.Login)
			site = sitemodel.Open(f.Page, domainName)
			site.NavigateToServers()
		})

		It("should add a new server [expand]", func() {
			siteServerPage := site.GetSiteServerPage()
			siteServer = siteServerPage.AddAWSServer()
		})

		It("should remove a new server [shrink]", func() {
			siteServerPage := site.GetSiteServerPage()
			siteServerPage.DeleteServer(siteServer)
		})
	})

	framework.RoboDescribe("Site delete operation [provisioner:aws][delete]", func() {
		It("should delete site", func() {
			ui.EnsureUser(f.Page, framework.Cluster.OpsCenterURL(), ctx.Login)
			By("navigating to opscenter")
			opscenter := opscentermodel.Open(f.Page, framework.Cluster.OpsCenterURL())
			By("trying to delete a site")
			opscenter.DeleteSite(ctx.ClusterName)
		})
	})
})
