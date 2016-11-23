package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	sitemodel "github.com/gravitational/robotest/e2e/model/ui/site"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func VerifySiteUpdate(f *framework.T) {

	PDescribe("Site Update", func() {
		ctx := framework.TestContext
		var domainName string
		var siteURL string

		BeforeEach(func() {
			domainName = ctx.ClusterName
			siteURL = framework.SiteURL()
		})

		It("should update site to the latest version", func() {
			ui.EnsureUser(f.Page, siteURL, ctx.Login)

			site := sitemodel.Open(f.Page, domainName)
			site.NavigateToSiteApp()

			appPage := site.GetSiteAppPage()
			newVersions := appPage.GetNewVersions()

			Expect(newVersions).NotTo(BeEmpty(), "should have at least 1 new version available")
			appPage.UpdateApp(&newVersions[0])
		})
	})
}
