package installer

import (
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/defaults"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type Installer struct {
	page       *web.Page
	domainName string
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
	installer := Installer{page: page, domainName: domainName}
	installer.navigateTo(url)
	return &installer
}

func Open(page *web.Page, URL string) *Installer {
	installer := Installer{page: page, domainName: framework.TestContext.ClusterName}
	installer.navigateTo(URL)
	return &installer
}

func (i *Installer) ProcessLicenseStepIfRequired(license string) {
	elems := i.page.FindByClass("grv-license")
	count, _ := elems.Count()
	if count > 0 {
		Expect(license).NotTo(BeEmpty(), "should have a valid license")
		Expect(elems.SendKeys(license)).To(Succeed())
		Expect(i.page.FindByClass("grv-installer-btn-new-site").Click()).To(Succeed(),
			"should input the license text")
		Eventually(i.page.FindByClass("grv-installer-warning"),
			defaults.FindTimeout).ShouldNot(BeFound(),
			"should not see an error about an invalid license")
	}
}

func (i *Installer) CreateAWSSite(domainName string) string {
	log.Infof("setting domain name")
	config := framework.TestContext.AWS
	Expect(i.IsCreateSiteStep()).To(BeTrue(), "should be on the select provisioner step")
	specifyDomainName(i.page, domainName)

	log.Infof("setting provisioner")
	Expect(i.page.FindByClass("--aws").Click()).To(Succeed())
	Expect(i.page.FindByName("aws_access_key").Fill(config.AccessKey)).To(Succeed())
	Expect(i.page.FindByName("aws_secret_key").Fill(config.SecretKey)).To(Succeed())
	Expect(i.page.FindByClass("grv-installer-btn-new-site").Click()).To(Succeed())
	Eventually(i.page.FindByClass("grv-installer-aws-region"), defaults.FindTimeout).Should(BeFound())

	log.Infof("setting region")
	ui.PauseForComponentJs()
	ui.SetDropdownValue(i.page, "grv-installer-aws-region", config.Region)

	log.Infof("setting key pair")
	ui.PauseForComponentJs()
	ui.SetDropdownValue(i.page, "grv-installer-aws-key-pair", config.KeyPair)

	log.Infof("setting VPC")
	ui.PauseForComponentJs()
	ui.SetDropdownValue(i.page, "grv-installer-aws-vpc", config.VPC)

	i.proceedToReqs()

	pageURL, err := i.page.URL()
	Expect(err).NotTo(HaveOccurred())
	return pageURL
}

func (i *Installer) CreateOnPremNewSite(domainName string) string {
	page := i.page
	Expect(i.IsCreateSiteStep()).To(BeTrue(), "should be on the select provisioner step")
	log.Infof("setting domain name")
	specifyDomainName(page, domainName)

	log.Infof("setting provisioner")
	Eventually(i.page.FindByClass("fa-check"), defaults.FindTimeout).Should(BeFound())
	Expect(page.FindByClass("--metal").Click()).To(Succeed())

	i.proceedToReqs()

	pageURL, err := page.URL()
	Expect(err).NotTo(HaveOccurred())
	return pageURL
}

func (i *Installer) PrepareOnPremNodes(dockerDevice string) {
	onpremProfiles := FindOnPremProfiles(i.page)
	Expect(len(onpremProfiles)).NotTo(Equal(0))
	numInstallNodes := 0
	for _, profile := range onpremProfiles {
		numInstallNodes = numInstallNodes + profile.Count
	}

	log.Infof("allocating %v nodes", numInstallNodes)
	provisioner := framework.Cluster.Provisioner()
	Expect(provisioner).NotTo(BeNil(), "expected valid provisioner for onprem installation")
	allocatedNodes, err := provisioner.NodePool().Allocate(numInstallNodes)
	Expect(err).NotTo(HaveOccurred(), "expected to allocate node(s)")
	Expect(allocatedNodes).To(HaveLen(numInstallNodes),
		fmt.Sprintf("expected to allocated %v nodes", numInstallNodes))

	log.Infof("executing the command on servers")
	index := 0
	for _, p := range onpremProfiles {
		nodesForProfile := allocatedNodes[index:p.Count]
		framework.RunAgentCommand(p.Command, nodesForProfile...)
		index = index + p.Count
	}

	log.Infof("waiting for agent report with the servers")
	for _, p := range onpremProfiles {
		Eventually(p.GetAgentServers, defaults.AgentServerTimeout).Should(
			HaveLen(p.Count))
	}

	log.Infof("configuring the servers with IPs")
	for _, p := range onpremProfiles {
		agentServers := p.GetAgentServers()
		for _, s := range agentServers {
			s.SetIPByInfra(provisioner)
			if s.NeedsDockerDevice() && dockerDevice != "" {
				s.SetDockerDevice(dockerDevice)
			}
		}
	}
}

func (i *Installer) GetAWSProfiles() []AWSProfile {
	var profiles []AWSProfile
	reqs := i.page.All(".grv-installer-provision-reqs-item")
	elements, err := reqs.Elements()
	Expect(err).NotTo(HaveOccurred())

	for index := range elements {
		profiles = append(profiles, createAWSProfile(i.page, index))
	}

	return profiles
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
	Eventually(i.IsInProgressStep, defaults.InstallStartTimeout).Should(BeTrue())
}

func (i *Installer) IsInstallCompleted() bool {
	count, _ := i.page.Find(".grv-installer-progress-result .fa-check").Count()
	return count != 0
}

func (i *Installer) IsInstallFailed() bool {
	count, _ := i.page.Find(".grv-installer-progress-result .fa-exclamation-triangle").Count()
	return count != 0
}

func (i *Installer) NeedsBandwagon() bool {
	needsBandwagon := false
	const jsTemplate = `		            
		var ver1x = window.reactor.evaluate(["sites", "%[1]v"]).getIn(["app", "manifest", "installer", "final_install_step", "service_name"]);
		var ver3x = window.reactor.evaluate(["sites", "%[1]v"]).getIn(["app", "manifest", "installer", "setupEndpoints"]);

		if(ver3x || ver1x){
			return true;
		}

		return false;			                        
	`
	js := fmt.Sprintf(jsTemplate, i.domainName)
	Expect(i.page.RunScript(js, nil, &needsBandwagon)).To(Succeed(), "should detect if bandwagon is required")
	return needsBandwagon
}

func (i *Installer) SelectFlavorByIndex(index int) {
	cssSelector := fmt.Sprintf(".grv-slider-value-desc:nth-child(%v) span", index)
	elem := i.page.First(cssSelector)
	Expect(elem).To(BeFound())
	Expect(elem.Click()).To(Succeed())
}

func (i *Installer) WaitForComplete() {
	Expect(i.IsInProgressStep()).To(BeTrue(), "should be in progress")
	installTimeout := defaults.InstallTimeout
	if framework.TestContext.Extensions.InstallTimeout != 0 {
		installTimeout = framework.TestContext.Extensions.InstallTimeout.Duration()
	}
	Eventually(func() bool {
		return i.IsInstallCompleted() || i.IsInstallFailed()
	}, installTimeout, defaults.InstallSuccessMessagePollInterval).Should(BeTrue(), "wait until timeout or install success/fail message")

	Expect(i.IsInstallFailed()).To(BeFalse(), "should not fail")
	Expect(i.IsInstallCompleted()).To(BeTrue(), "should be completed")
}

func (i *Installer) SelectFlavorByLabel(label string) int {
	Expect(i.IsRequirementsReviewStep()).To(BeTrue(), "should be on cluster requirements step")
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

func (i *Installer) proceedToReqs() {
	Expect(i.page.FindByClass("grv-installer-btn-new-site").Click()).To(Succeed())
	Eventually(i.page.FindByClass("grv-installer-provision-reqs"), defaults.ProvisionerSelectedTimeout).Should(BeFound())
}

func (i *Installer) navigateTo(URL string) {
	log.Infof("navigating to %q", URL)
	Expect(i.page.Navigate(URL)).To(Succeed())
	Eventually(i.page.FindByClass("grv-installer"), defaults.FindTimeout).Should(BeFound())
	ui.PauseForPageJs()
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
