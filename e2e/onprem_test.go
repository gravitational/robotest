/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/framework/defaults"
	"github.com/gravitational/robotest/e2e/uimodel"
	"github.com/gravitational/robotest/infra/terraform"

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
		if !installer.NeedsBandwagon(domainName) {
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
		siteEntryURL := makeSiteEntryURL(ctx, endpoints)

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
		log.Infof("connecting to cluster url: %v", siteEntryURL)
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

func makeSiteEntryURL(ctx *framework.TestContextType, endpoints []string) string {
	// Use the first allocated node to access the local site
	allocatedNodes := framework.Cluster.Provisioner().NodePool().AllocatedNodes()
	siteEntryURL := endpoints[0]

	if ctx.Provisioner.Type == "terraform" && ctx.Onprem.ClusterAddress != nil {
		switch ctx.Onprem.ClusterAddress.Type {
		case defaults.Public:
			// use public IP address for terraform provisioned nodes
			installNode := allocatedNodes[0]
			siteEntryURL = fmt.Sprintf("https://%v:%v", installNode.Addr(), defaults.GravityHTTPPort)
		case defaults.LoadBalancer:
			// use loadbalancer address and port defined in config or default gravity port instead
			port := defaults.GravityHTTPPort
			if ctx.Onprem.ClusterAddress.Port != 0 {
				port = ctx.Onprem.ClusterAddress.Port
			}
			state := framework.Cluster.Provisioner().State()
			switch specific := state.Specific.(type) {
			case *terraform.State:
				siteEntryURL = fmt.Sprintf("https://%v:%v", specific.LoadBalancerAddr, port)
			default:
				Fail("unreachable")
			}
		}
	}

	return siteEntryURL
}
