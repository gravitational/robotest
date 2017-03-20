package installer

import (
	"encoding/json"
	"fmt"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/defaults"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type Installer struct {
	page *web.Page
}

type serverProfile struct {
	// Nodes specifies the number of nodes to provision
	Nodes int
}

func OpenWithSite(page *web.Page, domainName string) *Installer {
	installerPath := fmt.Sprintf("/web/installer/site/%v", domainName)
	url, err := page.URL()
	Expect(err).NotTo(HaveOccurred())
	url = framework.URLPathFromString(url, installerPath)

	return Open(page, url)
}

func Open(page *web.Page, URL string) *Installer {
	By(fmt.Sprintf("navigating to %q", URL))
	Expect(page.Navigate(URL)).To(Succeed())
	Eventually(page.FindByClass("grv-installer"), defaults.FindTimeout).Should(BeFound())

	ui.PauseForPageJs()

	return &Installer{page: page}
}

func (i *Installer) FillOutLicenseIfRequired(license string) {
	elems := i.page.FindByClass("grv-license")
	count, _ := elems.Count()
	if count > 0 {
		Expect(license).NotTo(BeEmpty(), "should have a valid license")
		Expect(elems.SendKeys(license)).To(Succeed())
		Expect(i.page.FindByClass("grv-installer-btn-new-site").Click()).To(Succeed(),
			"should input the license text")
		Eventually(i.page.FindByClass("grv-installer-warning"),
			defaults.FindTimeout, defaults.SelectionPollInterval).ShouldNot(BeFound(),
			"should not see an error about an invalid license")
	}
}

func (i *Installer) CreateAWSSite(domainName string, config framework.AWSConfig) string {
	By("setting domain name")
	page := i.page
	specifyDomainName(page, domainName)

	By("setting provisioner")
	Expect(page.FindByClass("--aws").Click()).To(Succeed())
	Expect(page.FindByName("aws_access_key").Fill(config.AccessKey)).To(Succeed())
	Expect(page.FindByName("aws_secret_key").Fill(config.SecretKey)).To(Succeed())
	Expect(page.FindByClass("grv-installer-btn-new-site").Click()).To(Succeed())
	Eventually(page.FindByClass("grv-installer-aws-region"), defaults.FindTimeout).Should(BeFound())

	ui.PauseForComponentJs()
	By("setting region")
	ui.SetDropdownValue(page, "grv-installer-aws-region", config.Region)

	ui.PauseForComponentJs()
	By("setting key pair")
	ui.SetDropdownValue(page, "grv-installer-aws-key-pair", config.KeyPair)

	ui.PauseForComponentJs()
	By("setting VPC")
	ui.SetDropdownValue(page, "grv-installer-aws-vpc", config.VPC)

	i.proceedToReqs()

	pageURL, err := page.URL()
	Expect(err).NotTo(HaveOccurred())
	return pageURL
}

func (i *Installer) CreateOnPremNewSite(domainName string) string {
	page := i.page
	By("setting domain name")
	specifyDomainName(page, domainName)

	By("setting provisioner")
	Eventually(page.FindByClass("fa-check"), defaults.FindTimeout).Should(BeFound())
	Expect(page.FindByClass("--metal").Click()).To(Succeed())

	i.proceedToReqs()

	pageURL, err := page.URL()
	Expect(err).NotTo(HaveOccurred())
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
	button := i.page.Find(".grv-installer-footer .btn-primary")
	Expect(button).To(BeFound())
	Expect(button.Click()).To(Succeed())
	Eventually(i.IsInProgressStep, defaults.StartInstallTimeout).Should(BeTrue())
}

func (i *Installer) IsInstallCompleted() bool {
	count, _ := i.page.Find(".grv-installer-progress-result .fa-check").Count()
	return count != 0
}

func (i *Installer) SelectFlavorByIndex(index int) {
	cssSelector := fmt.Sprintf(".grv-slider-value-desc:nth-child(%v) span", index)
	elem := i.page.First(cssSelector)
	Expect(elem).To(BeFound())
	Expect(elem.Click()).To(Succeed())
}

func (i *Installer) SelectFlavorByLabel(label string) int {
	labels := i.page.All(".grv-slider-value-desc span")
	Expect(labels).To(BeFound())

	elems, err := labels.Elements()
	Expect(err).NotTo(HaveOccurred())

	for _, elem := range elems {
		text, err := elem.GetText()
		Expect(err).NotTo(HaveOccurred())
		if text == label {
			Expect(elem.Click()).To(Succeed())
			return getServerCountFromSelectedProfile(i.page)
		}
	}
	framework.Failf("no flavor matches the specified label %q", label)
	return 0 // unreachable
}

func getServerCountFromSelectedProfile(page *web.Page) int {
	const script = `
            var getter = [ ["installer_provision", "profilesToProvision"], profiles => {
                return profiles.map(profile => {
                    return {
                        Nodes: profile.get("count"),
                    }
                }).toJS();
            }];

            var data = window.reactor.evaluate(getter)
            return JSON.stringify(data);
        `
	var profiles map[string]serverProfile
	var profileBytes string

	Expect(page.RunScript(script, nil, &profileBytes)).To(Succeed())
	Expect(json.Unmarshal([]byte(profileBytes), &profiles)).To(Succeed())

	var count int
	for _, profile := range profiles {
		count = count + profile.Nodes
	}
	return count
}

func specifyDomainName(page *web.Page, domainName string) {
	Eventually(page.FindByName("domainName"), defaults.FindTimeout).Should(BeFound())
	Expect(page.FindByName("domainName").Fill(domainName)).To(Succeed())
	Eventually(page.FindByClass("fa-check"), defaults.FindTimeout).Should(BeFound())
}

func (i *Installer) proceedToReqs() {
	Expect(i.page.FindByClass("grv-installer-btn-new-site").Click()).To(Succeed())
	Eventually(i.page.FindByClass("grv-installer-provision-reqs"), defaults.ProvisionerSelectedTimeout).Should(BeFound())
}
