package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	sitemodel "github.com/gravitational/robotest/e2e/model/ui/site"

	libdefaults "github.com/gravitational/robotest/lib/defaults"
	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func VerifySiteUpdate(f *framework.T) {

	framework.RoboDescribe("Site Update", func() {
		ctx := framework.TestContext
		var domainName string
		var siteURL string

		BeforeEach(func() {
			domainName = ctx.ClusterName
			siteURL = framework.SiteURL()
		})

		It("should update site to the latest version", func() {
			By("uploading new application into site")
			if ctx.Onprem.InstallerURL == "" {
				// Upload a new version to Ops Center
				// TODO: remove the fake version at cleanup/teardown
				framework.FakeUpdateApplication()
			} else {

				framework.UpdateApplicationWithInstaller()
			}

			By("updating site to the latest application version")
			ui.EnsureUser(f.Page, siteURL, ctx.Login)

			site := sitemodel.Open(f.Page, domainName)
			site.NavigateToSiteApp()

			appPage := site.GetSiteAppPage()
			newVersions := appPage.GetNewVersions()

			Expect(newVersions).NotTo(BeEmpty(), "should have at least 1 new version available")
			appPage.UpdateApp(newVersions[0])

			By("check update is finished")
			// Here have to login again, because update of gravity-site app will logout us
			// Check for right provider before login

			err := wait.Retry(libdefaults.RetryDelay, libdefaults.RetryAttempts, func() error {
				err := ui.IsLoginPageFound(f.Page, siteURL, ctx.Login)
				if err != nil {
					log.Debug(trace.DebugReport(err))
				}
				return trace.Wrap(err)
			})
			Expect(err).NotTo(HaveOccurred())

			ui.EnsureUser(f.Page, siteURL, ctx.Login)

			site = sitemodel.Open(f.Page, domainName)
			site.NavigateToSiteApp()

			appPage = site.GetSiteAppPage()
			appPage.CheckUpdateApp(newVersions[0])
		})
	})
}
