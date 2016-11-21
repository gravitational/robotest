package e2e

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/specs"

	. "github.com/onsi/ginkgo"
)

var _ = framework.RoboDescribe("Onprem Integration Test", func() {
	f := framework.New()
	ctx := framework.TestContext

	BeforeEach(func() {
		ui.EnsureUser(f.Page, framework.InstallerURL(),
			ctx.Login.Username, ctx.Login.Password, ui.WithGoogle)
	})

	specs.VerifyOnpremInstall(f)
})
