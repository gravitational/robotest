package installer

import (
	"time"

	"github.com/gravitational/robotest/e2e/ui/installer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
)

var (
	defaultTimeout = 20 * time.Second
)

func CreateNewSite(page *agouti.Page, startURL string, domainName string) {
	inst := installer.OpenInstaller(page, startURL)
	By("entering domain name")
	Eventually(inst.IsCreateSiteStep, defaultTimeout).Should(BeTrue())
	inst.CreateOnPremNewSite(domainName)
}

func MeetRequrements(page *agouti.Page, domainName string) {
	inst := installer.OpenInstallerWithSite(page, domainName)
	Expect(inst.IsRequirementsReviewStep()).To(BeTrue())

	By("selecting a flavor")
	inst.SelectFlavor(2)

	By("veryfing requirements")
	profiles := installer.FindOnPremProfiles(page)
	Expect(len(profiles)).To(Equal(1))

	By("waiting for agent report with the servers")
	Eventually(profiles[0].GetServers, 10*time.Minute).Should(HaveLen(1))

	By("veryfing that server has IP")
	server := profiles[0].GetServers()[0]
	ips := server.GetIPs()
	Expect(len(ips) == 2).To(BeTrue())

	By("starting an installation")
	inst.StartInstallation()

	time.Sleep(10 * time.Second)
}

func WaitForComplete(page *agouti.Page, domainName string) {
	By("waiting for success notification")
	inst := installer.OpenInstallerWithSite(page, domainName)
	Expect(inst.IsInProgressStep()).To(BeTrue())
	Eventually(inst.IsInstallCompleted, 10*time.Minute).Should(BeTrue())

	By("clicking on continue")
	inst.ProceedToSite()
}
