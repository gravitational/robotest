package installer

import (
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/uimodel/defaults"
	"github.com/gravitational/robotest/e2e/uimodel/utils"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

// Installer is installer ui model
type Installer struct {
	page *web.Page
}

// Open creates installer ui model and navigates to given URL
func Open(page *web.Page, URL string) Installer {
	installer := Installer{page: page}
	installer.navigateTo(URL)
	return installer
}

// ProcessLicenseStepIfRequired handles installer step which requires a license
func (i *Installer) ProcessLicenseStepIfRequired(license string) {
	elems := i.page.FindByClass("grv-license")
	count, _ := elems.Count()
	if count > 0 {
		Expect(license).NotTo(BeEmpty(), "should have a valid license")
		Expect(elems.SendKeys(license)).To(Succeed())
		Expect(i.page.FindByClass("grv-installer-btn-new-site").Click()).
			To(Succeed(), "should input the license text")
		Eventually(i.page.FindByClass("grv-installer-warning"), defaults.FindTimeout).
			ShouldNot(BeFound(), "should not see an error about an invalid license")
	}
}

// CreateSiteWithAWS initilizes cluster install operation using AWS
func (i *Installer) CreateSiteWithAWS(domainName string) string {
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
	utils.PauseForComponentJs()
	utils.SetDropdownValue(i.page, "grv-installer-aws-region", config.Region)

	log.Infof("setting key pair")
	utils.PauseForComponentJs()
	utils.SetDropdownValue(i.page, "grv-installer-aws-key-pair", config.KeyPair)

	log.Infof("setting VPC")
	utils.PauseForComponentJs()
	utils.SetDropdownValue(i.page, "grv-installer-aws-vpc", config.VPC)

	i.proceedToReqs()

	pageURL, err := i.page.URL()
	Expect(err).NotTo(HaveOccurred())
	return pageURL
}

// CreateSiteWithOnPrem initilizes cluster install operation using OnPrem
func (i *Installer) CreateSiteWithOnPrem(domainName string) string {
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

// PrepareOnPremNodes sets parameters for each found node
func (i *Installer) PrepareOnPremNodes(dockerDevice string) {
	onpremProfiles := i.GetOnPremProfiles()
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

// GetOnPremProfiles returns a list of onprem profiles
func (i *Installer) GetOnPremProfiles() []OnPremProfile {
	log.Infof("getting onprem profiles")
	elements, _ := i.page.All(".grv-installer-provision-reqs-item").Elements()
	var profiles = []OnPremProfile{}
	for index := range elements {
		profiles = append(profiles, createProfile(i.page, index))
	}

	return profiles
}

// GetAWSProfiles returns a list of aws profiles
func (i *Installer) GetAWSProfiles() []AWSProfile {
	log.Infof("getting AWS profiles")
	var profiles []AWSProfile
	elements, err := i.page.All(".grv-installer-provision-reqs-item").Elements()
	Expect(err).NotTo(HaveOccurred())
	for index := range elements {
		profiles = append(profiles, createAWSProfile(i.page, index))
	}

	return profiles
}

// ProceedToSite proceeds to the cluster site once installation is completed.
func (i *Installer) ProceedToSite() {
	log.Infof("trying to proceed to site")
	Expect(i.page.Find(".grv-installer-progress-result .btn-primary").Click()).To(Succeed())
}

// StartInstallation starts install operation
func (i *Installer) StartInstallation() {
	log.Infof("clicking on start installation")
	button := i.page.Find(".grv-installer-footer .btn-primary")
	Expect(button).To(BeFound())
	Expect(button.Click()).To(Succeed())
	Eventually(func() bool {
		return i.IsInProgressStep() || i.IsWarningVisible()
	}, defaults.InstallStartTimeout).Should(BeTrue())

	Expect(i.IsInProgressStep()).To(BeTrue(), "should successfully start an installation")
}

// IsCreateSiteStep checks if installer is an initial step
func (i *Installer) IsCreateSiteStep() bool {
	log.Infof("checking if on the create new cluster step")
	count, _ := i.page.FindByClass("grv-installer-fqdn").Count()
	return count != 0
}

// IsInProgressStep checks if installer is in progress
func (i *Installer) IsInProgressStep() bool {
	log.Infof("checking if installation is in progress")
	count, _ := i.page.FindByClass("grv-installer-progres-indicator").Count()
	return count != 0
}

// IsRequirementsReviewStep checks if installer is on the requirements step
func (i *Installer) IsRequirementsReviewStep() bool {
	count, _ := i.page.FindByClass("grv-installer-provision-reqs").Count()
	return count != 0
}

// IsWarningVisible checks if installer has any warnings visible
func (i *Installer) IsWarningVisible() bool {
	log.Infof("checking if warning icon is present")
	count, _ := i.page.Find(".grv-installer-attemp-message .--warning").Count()
	return count != 0
}

// IsInstallCompleted checks if install operation has been completed
func (i *Installer) IsInstallCompleted() bool {
	log.Infof("checking if installation is completed")
	count, _ := i.page.Find(".grv-installer-progress-result .fa-check").Count()
	return count != 0
}

// IsInstallFailed checks if install operation failed
func (i *Installer) IsInstallFailed() bool {
	log.Infof("checking if installation is failed")
	count, _ := i.page.Find(".grv-installer-progress-result .fa-exclamation-triangle").Count()
	return count != 0
}

// NeedsBandwagon checks if installation has a bandwagon step
func (i *Installer) NeedsBandwagon(domainName string) bool {
	log.Infof("checking if bandwagon is required")
	needsBandwagon := false
	const jsTemplate = `		            
		var ver1x = window.reactor.evaluate(["sites", "%[1]v"]).getIn(["app", "manifest", "installer", "final_install_step", "service_name"]);
		var ver3x = window.reactor.evaluate(["sites", "%[1]v"]).getIn(["app", "manifest", "installer", "setupEndpoints"]);

		if(ver3x || ver1x){
			return true;
		}

		return false;			                        
	`
	js := fmt.Sprintf(jsTemplate, domainName)
	Expect(i.page.RunScript(js, nil, &needsBandwagon)).To(Succeed(), "should detect if bandwagon is required")
	return needsBandwagon
}

// SelectFlavorByIndex selects flavor by index
func (i *Installer) SelectFlavorByIndex(index int) {
	log.Infof("selecting a flavor")
	cssSelector := fmt.Sprintf(".grv-slider-value-desc:nth-child(%v) span", index)
	elem := i.page.First(cssSelector)
	Expect(elem).To(BeFound())
	Expect(elem.Click()).To(Succeed())
}

// SelectFlavorByLabel selects flavor by label
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

// WaitForComplete waits for ongoing install operation to be completed
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

func (i *Installer) proceedToReqs() {
	log.Infof("trying to init install operation")
	Expect(i.page.FindByClass("grv-installer-btn-new-site").Click()).To(Succeed())
	Eventually(func() bool {
		return i.IsWarningVisible() || i.IsRequirementsReviewStep() || utils.HasValidationErrors(i.page)
	}, defaults.InstallCreateClusterTimeout).Should(BeTrue())
	Expect(i.IsRequirementsReviewStep()).To(BeTrue(), "should be on requirements step")
}

func (i *Installer) navigateTo(URL string) {
	Expect(i.page.Navigate(URL)).To(Succeed())
	Eventually(func() bool {
		return utils.IsInstaller(i.page) || utils.IsErrorPage(i.page)
	}, defaults.AppLoadTimeout).Should(BeTrue())

	Expect(utils.IsInstaller(i.page)).To(BeTrue(), "valid installer page")
	utils.PauseForPageJs()
}

type serverProfile struct {
	// Nodes specifies the number of nodes to provision
	Nodes int
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
