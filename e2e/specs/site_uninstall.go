package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"

	. "github.com/onsi/ginkgo"
)

func SiteUninstall(f *framework.T) {
	var _ = framework.RoboDescribe("Site Uninstall", func() {
		ctx := framework.TestContext

		It("should be able to uninstall site", func() {
			ui.EnsureUser(f.Page, framework.Cluster.OpsCenterURL(), ctx.Login)

			By("deleting site")
			ui.DeleteSite(f.Page, ctx.ClusterName)
		})
	})
}
