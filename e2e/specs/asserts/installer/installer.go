package installer

import (
	"time"

	"github.com/gravitational/robotest/e2e/ui/installer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
)

func WaitForComplete(page *agouti.Page, domainName string) {
	inst := installer.OpenWithSite(page, domainName)

	By("checking if on in progress screen")
	Expect(inst.IsInProgressStep()).To(BeTrue())

	Eventually(inst.IsInstallCompleted, 20*time.Minute).Should(
		BeTrue(), "wait until timeout or install success message")

	By("clicking on continue")
	inst.ProceedToSite()
}
