package installer

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui/defaults"
	installermodel "github.com/gravitational/robotest/e2e/model/ui/installer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
)

func WaitForComplete(page *web.Page, domainName string) {
	installer := installermodel.OpenWithSite(page, domainName)

	By("verifying that the progress screen is active")
	Expect(installer.IsInProgressStep()).To(BeTrue())

	installTimeout := defaults.InstallTimeout
	if framework.TestContext.Extensions.InstallTimeout != 0 {
		installTimeout = framework.TestContext.Extensions.InstallTimeout.Duration()
	}
	Eventually(func() bool {
		return installer.IsInstallCompleted() || installer.IsInstallFailed()
	}, installTimeout, defaults.PollInterval).Should(BeTrue(), "wait until timeout or install success/fail message")

	Expect(installer.IsInstallFailed()).To(BeFalse())
	Expect(installer.IsInstallCompleted()).To(BeTrue())

	By("clicking on continue")
	installer.ProceedToSite()
}
