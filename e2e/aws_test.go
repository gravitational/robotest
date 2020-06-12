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
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/uimodel"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = framework.RoboDescribe("AWS Integration Test", func() {
	f := framework.New()
	ctx := framework.TestContext

	It("should provision a new cluster [provisioner:aws,install]", func() {
		domainName := ctx.ClusterName
		ui := uimodel.InitWithUser(f.Page, framework.InstallerURL())
		installer := ui.GoToInstaller(framework.InstallerURL())

		By("filling out license text field if required")
		installer.ProcessLicenseStepIfRequired(ctx.License)
		installer.InitAWSInstallation(domainName)

		By("selecting a flavor")
		installer.SelectFlavorByLabel(ctx.FlavorLabel)
		profiles := installer.GetAWSProfiles()
		Expect(len(profiles)).To(BeNumerically(">", 0), "expect at least 1 profile")

		By("setting up AWS instance types")
		for _, p := range profiles {
			p.SetInstanceType(ctx.AWS.InstanceType)
		}

		By("starting an installation")
		installer.StartInstallation()

		By("waiting until install is completed or failed")
		installer.WaitForCompletion()

		if installer.NeedsBandwagon(domainName) {
			By("navigating to bandwagon step")
			bandwagon := ui.GoToBandwagon(domainName)
			By("submitting bandwagon form")
			ctx.Bandwagon.RemoteAccess = ctx.ForceRemoteAccess || !ctx.Wizard
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

	It("should add and remove a server [provisioner:aws,expand-shrink]", func() {
		ui := uimodel.InitWithUser(f.Page, framework.SiteURL())
		site := ui.GoToSite(ctx.ClusterName)
		siteServerPage := site.GoToServers()
		newServer := siteServerPage.AddAWSServer()
		siteServerPage.DeleteServer(newServer)
	})

	It("should delete site [provisioner:aws,delete]", func() {
		ui := uimodel.InitWithUser(f.Page, framework.Cluster.OpsCenterURL())
		opscenter := ui.GoToOpsCenter()
		opscenter.DeleteSite(ctx.ClusterName)
	})
})
