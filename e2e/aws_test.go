package e2e

import (
	"github.com/gravitational/robotest/e2e/framework"

	"github.com/gravitational/robotest/e2e/uimodel"
	sitemodel "github.com/gravitational/robotest/e2e/uimodel/site"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = framework.RoboDescribe("AWS Integration Test", func() {
	f := framework.New()
	ctx := framework.TestContext

	var ui uimodel.UI

	BeforeEach(func() {
		ui = uimodel.Init(f.Page)
	})

	framework.RoboDescribe("Provisioning a new cluster [provisioner:aws][install]", func() {
		It("should provision a new cluster", func() {
			domainName := ctx.ClusterName
			ui.EnsureUser(framework.InstallerURL())
			installer := ui.GoToInstaller(framework.InstallerURL())

			By("filling out license text field if required")
			installer.ProcessLicenseStepIfRequired(ctx.License)
			installer.CreateSiteWithAWS(domainName)

			By("selecting a flavor")
			installer.SelectFlavorByLabel(ctx.FlavorLabel)
			profiles := installer.GetAWSProfiles()
			Expect(len(profiles)).To(Equal(1), "should verify required node number")

			By("setting up AWS instance types")
			for _, p := range profiles {
				p.SetInstanceType(ctx.AWS.InstanceType)
			}

			By("starting an installation")
			installer.StartInstallation()

			By("waiting until install is completed or failed")
			installer.WaitForComplete()

			if installer.NeedsBandwagon(domainName) == true {
				By("navigating to bandwagon step")
				bandwagon := ui.GoToBandwagon(domainName)
				By("submitting bandwagon form")
				enableRemoteAccess := ctx.ForceRemoteAccess || !ctx.Wizard
				ctx.Bandwagon.RemoteAccess = enableRemoteAccess
				bandwagon.SubmitForm(ctx.Bandwagon)

				By("navigating to a site and reading endpoints")
				site := ui.GoToSite(domainName)
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
			ui.EnsureUser(framework.SiteURL())
			site = ui.GoToSite(ctx.ClusterName)
		})

		It("should add a new server [expand]", func() {
			siteServerPage := site.GoToServers()
			siteServer = siteServerPage.AddAWSServer()
		})

		It("should remove a new server [shrink]", func() {
			siteServerPage := site.GoToServers()
			siteServerPage.DeleteServer(siteServer)
		})
	})

	framework.RoboDescribe("Site delete operation [provisioner:aws][delete]", func() {
		It("should delete site", func() {
			By("openning opscenter")
			opscenter := ui.GoToOpsCenter(framework.Cluster.OpsCenterURL())
			By("trying to delete a site")
			opscenter.DeleteSite(ctx.ClusterName)
		})
	})
})
