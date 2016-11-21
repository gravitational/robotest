package installer

import (
	installermodel "github.com/gravitational/robotest/e2e/model/ui/installer"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
)

func WaitForComplete(page *web.Page, domainName string) {
	installer := installermodel.OpenWithSite(page, domainName)

	By("verifying that the progress screen is active")
	Expect(installer.IsInProgressStep()).To(BeTrue())

	Eventually(installer.IsInstallCompleted, defaults.InstallTimeout, defaults.PollInterval).Should(
		BeTrue(), "wait until timeout or install success message")

	By("clicking on continue")
	installer.ProceedToSite()
}
