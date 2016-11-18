package installer

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/ui/common"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type Installer struct {
	page *agouti.Page
}

func OpenWithSite(page *agouti.Page, domainName string) *Installer {
	urlPrefix := fmt.Sprintf("/web/installer/site/%v", domainName)
	r, _ := regexp.Compile("/web/.*")
	url, _ := page.URL()
	url = r.ReplaceAllString(url, urlPrefix)

	return Open(page, url)
}

func Open(page *agouti.Page, URL string) *Installer {
	By("Navigating to installer screen")
	Expect(page.Navigate(URL)).To(Succeed())
	Eventually(page.FindByClass("grv-installer"), defaults.FindTimeout).Should(BeFound())

	time.Sleep(1000 * time.Millisecond)

	return &Installer{page: page}
}

func (i *Installer) CreateAwsSite(domainName string, config framework.AWSConfig) string {
	By("Setting deployment name")
	page := i.page
	specifyDomainName(page, domainName)

	By("Setting provisioner")
	Expect(page.FindByClass("--aws").Click()).To(Succeed())
	Expect(page.FindByName("aws_access_key").Fill(config.AccessKey)).To(Succeed())
	Expect(page.FindByName("aws_secret_key").Fill(config.SecretKey)).To(Succeed())
	Expect(page.FindByClass("grv-installer-btn-new-site").Click()).To(Succeed())
	Eventually(page.FindByClass("grv-installer-aws-region"), defaults.FindTimeout).Should(BeFound())

	pause()
	By("Setting region")
	common.SetDropDownValue(page, "grv-installer-aws-region", config.Region)

	pause()
	By("Setting key pair")
	common.SetDropDownValue(page, "grv-installer-aws-key-pair", config.KeyPair)

	pause()
	By("Setting VPC")
	common.SetDropDownValue(page, "grv-installer-aws-vpc", config.VPC)

	i.proceedToReqs()

	pageURL, _ := page.URL()
	return pageURL
}

func (i *Installer) CreateOnPremNewSite(domainName string) string {
	page := i.page
	By("Setting deployment name")
	specifyDomainName(page, domainName)

	By("Setting provisioner")
	Eventually(page.FindByClass("fa-check"), defaults.FindTimeout).Should(BeFound())
	Expect(page.FindByClass("--metal").Click()).To(Succeed())

	i.proceedToReqs()

	pageURL, _ := page.URL()
	return pageURL
}

func (i *Installer) ProceedToSite() {
	Expect(i.page.Find(".grv-installer-progress-result .btn-primary").Click()).To(Succeed())
}

func (i *Installer) IsCreateSiteStep() bool {
	count, _ := i.page.FindByClass("grv-installer-fqdn").Count()
	return count != 0
}

func (i *Installer) IsInProgressStep() bool {
	count, _ := i.page.FindByClass("grv-installer-progres-indicator").Count()
	return count != 0
}

func (i *Installer) IsRequirementsReviewStep() bool {
	count, _ := i.page.FindByClass("grv-installer-provision-reqs").Count()
	return count != 0
}

func (i *Installer) StartInstallation() {
	btn := i.page.Find(".grv-installer-footer .btn-primary")
	Expect(btn).To(BeFound())
	Expect(btn.Click()).To(Succeed())
	Eventually(i.IsInProgressStep, defaults.FindTimeout).Should(BeTrue())
}

func (i *Installer) IsInstallCompleted() bool {
	count, _ := i.page.Find(".grv-installer-progress-result .fa-check").Count()
	return count != 0
}

func (i *Installer) SelectFlavor(index int) {
	cssSelector := fmt.Sprintf(".grv-slider-value-desc:nth-child(%v) span", index)
	elem := i.page.First(cssSelector)
	Expect(elem).To(BeFound())
	Expect(elem.Click()).To(Succeed())
}

func specifyDomainName(page *agouti.Page, domainName string) {
	Eventually(page.FindByName("domainName"), defaults.FindTimeout).Should(BeFound())
	Expect(page.FindByName("domainName").Fill(domainName)).To(Succeed())
	Eventually(page.FindByClass("fa-check"), defaults.FindTimeout).Should(BeFound())
}

func (i *Installer) proceedToReqs() {
	Expect(i.page.FindByClass("grv-installer-btn-new-site").Click()).To(Succeed())
	Eventually(i.page.FindByClass("grv-installer-provision-reqs"), defaults.FindTimeout).Should(BeFound())
}

func pause() {
	time.Sleep(100 * time.Millisecond)
}
