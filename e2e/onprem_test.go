package e2e

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/framework/defaults"
	"github.com/gravitational/robotest/e2e/uimodel"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var _ = framework.RoboDescribe("Onprem Integration Test", func() {
	f := framework.New()
	ctx := framework.TestContext

	It("should provision a new cluster [provisioner:onprem,install]", func() {
		By("navigating to installer step")
		domainName := ctx.ClusterName
		ui := uimodel.InitWithUser(f.Page, framework.InstallerURL())
		installer := ui.GoToInstaller(framework.InstallerURL())

		By("filling out license text field if required")
		installer.ProcessLicenseStepIfRequired(ctx.License)

		By("selecting a provisioner")
		installer.InitOnPremInstallation(domainName)

		By("selecting a flavor and allocating the nodes")
		installer.SelectFlavorByLabel(ctx.FlavorLabel)
		installer.PrepareOnPremNodes(ctx.Onprem.DockerDevice)

		By("starting an installation")
		installer.StartInstallation()

		By("waiting until install is completed")
		installer.WaitForCompletion()

		By("checking for bandwagon step")
		if installer.NeedsBandwagon(domainName) == false {
			ui.GoToSite(domainName)
			return
		}

		By("navigating to bandwagon step")
		installer.ProceedToSite()
		bandwagon := ui.GoToBandwagon(domainName)

		By("submitting bandwagon form")
		ctx.Bandwagon.RemoteAccess = ctx.ForceRemoteAccess || !ctx.Wizard
		bandwagon.SubmitForm(ctx.Bandwagon)

		By("navigating to a site and reading endpoints")
		site := ui.GoToSite(domainName)
		endpoints := site.GetEndpoints()
		endpoints = filterGravityEndpoints(endpoints)
		Expect(len(endpoints)).To(BeNumerically(">", 0), "expected at least one application endpoint")

		By("using local application endpoint")
		// Use the first allocated node to access the local site
		allocatedNodes := framework.Cluster.Provisioner().NodePool().AllocatedNodes()
		siteEntryURL := endpoints[0]
		// For terraform, use public install node address
		// terraform nodes are provisioned only with a single private network interface
		if ctx.Provisioner == "terraform" {
			installNode := allocatedNodes[0]
			siteEntryURL = fmt.Sprintf("https://%v:%v", installNode.Addr(), defaults.GravityHTTPPort)
		}
		log.Infof("connecting to cluster url: %v", siteEntryURL)

		login := framework.Login{
			Username: framework.TestContext.Bandwagon.Email,
			Password: framework.TestContext.Bandwagon.Password,
		}
		serviceLogin := &framework.ServiceLogin{
			Username: login.Username,
			Password: login.Password}

		By("login in with bandwagon user credentials")
		framework.UpdateSiteEntry(siteEntryURL, login, serviceLogin)
		// login using local cluster endpoint
		ui = uimodel.InitWithUser(f.Page, framework.SiteURL())
		ui.GoToSite(domainName)
	})

	It("should add and remove a server [provisioner:onprem,expand-shrink]", func() {
		ui := uimodel.InitWithUser(f.Page, framework.SiteURL())
		site := ui.GoToSite(ctx.ClusterName)
		siteServerPage := site.GoToServers()
		newSiteServer := siteServerPage.AddOnPremServer()
		siteServerPage.DeleteServer(newSiteServer)
	})

	It("should update site to the latest version [provisioner:onprem,update]", func() {
		By("uploading new application into site")
		if ctx.Onprem.InstallerURL == "" {
			// Upload a new version to Ops Center
			// TODO: remove the fake version at cleanup/teardown
			framework.FakeUpdateApplication()
		} else {
			framework.UpdateApplicationWithInstaller()
		}

		By("trying to update the site to the latest application version")
		ui := uimodel.InitWithUser(f.Page, framework.SiteURL())
		site := ui.GoToSite(ctx.ClusterName)
		site.UpdateWithLatestVersion()
	})
})

func filterGravityEndpoints(endpoints []string) []string {
	var siteEndpoints []string
	for _, v := range endpoints {
		if strings.Contains(v, strconv.Itoa(defaults.GravityHTTPPort)) {
			siteEndpoints = append(siteEndpoints, v)
		}
	}

	return siteEndpoints
}
