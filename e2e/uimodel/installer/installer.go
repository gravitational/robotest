package installer

import (
	"encoding/json"
	"fmt"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/uimodel/defaults"
	"github.com/gravitational/robotest/e2e/uimodel/utils"

	log "github.com/Sirupsen/logrus"
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
		log.Infof("trying to complete license step")
		Expect(license).NotTo(BeEmpty(), "should have a valid license")
		Expect(elems.SendKeys(license)).To(Succeed())
		Expect(i.page.FindByClass("grv-installer-btn-new-site").Click()).
			To(Succeed(), "should input the license text")
		Eventually(i.page.FindByClass("grv-installer-warning"), defaults.FindTimeout).
			ShouldNot(BeFound(), "should be no warnings")
	}
}

// InitAWSInstallation initilizes cluster install operation using AWS
func (i *Installer) InitAWSInstallation(domainName string) {
	log.Infof("trying to initialize AWS install operation")
	Expect(i.IsCreateSiteStep()).To(BeTrue())
	specifyDomainName(i.page, domainName)

	log.Infof("providing AWS keys")
	config := framework.TestContext.AWS
	Expect(i.page.FindByClass("--aws").Click()).To(Succeed())
	Expect(i.page.FindByName("aws_access_key").Fill(config.AccessKey)).To(Succeed())
	Expect(i.page.FindByName("aws_secret_key").Fill(config.SecretKey)).To(Succeed())
	Expect(i.page.FindByClass("grv-installer-btn-new-site").Click()).To(Succeed())
	Eventually(func() bool {
		return utils.IsFound(i.page, ".grv-installer-aws-region") || i.IsWarningVisible()
	}, defaults.AjaxCallTimeout).Should(BeTrue(), "should accept AWS keys")

	Expect(i.IsWarningVisible()).To(BeFalse(), "should be no warnings")
	utils.PauseForComponentJs()

	log.Infof("setting region")
	utils.SetDropdownValue(i.page, "grv-installer-aws-region", config.Region)

	log.Infof("setting key pair")
	utils.SetDropdownValue(i.page, "grv-installer-aws-key-pair", config.KeyPair)

	log.Infof("setting VPC")
	utils.SetDropdownValue(i.page, "grv-installer-aws-vpc", config.VPC)
	i.proceedToReqs()
}

// InitOnPremInstallation initilizes cluster install operation using OnPrem
func (i *Installer) InitOnPremInstallation(domainName string) {
	log.Infof("trying to initialize onprem install operation")
	Expect(i.IsCreateSiteStep()).To(BeTrue())
	specifyDomainName(i.page, domainName)

	log.Infof("setting provisioner")
	Expect(i.page.FindByClass("--metal").Click()).To(Succeed())
	i.proceedToReqs()
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
		nodesForProfile := allocatedNodes[index : index+p.Count]
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
	log.Info("getting onprem profiles")
	elements, _ := i.page.All(".grv-installer-provision-reqs-item").Elements()
	var profiles = []OnPremProfile{}
	for index := range elements {
		profiles = append(profiles, createProfile(i.page, index))
	}

	return profiles
}

// GetAWSProfiles returns a list of aws profiles
func (i *Installer) GetAWSProfiles() []AWSProfile {
	log.Info("getting AWS profiles")
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
	log.Info("trying to proceed to site")
	Expect(i.page.Find(".grv-installer-progress-result .btn-primary").Click()).To(Succeed())
}

// StartInstallation starts install operation
func (i *Installer) StartInstallation() {
	log.Info("clicking on start installation")
	button := i.page.Find(".grv-installer-footer .btn-primary")
	Expect(button).To(BeFound())
	Expect(button.Click()).To(Succeed())
	Eventually(func() bool {
		return i.IsInProgressStep() || i.IsWarningVisible()
	}, defaults.InstallStartTimeout).Should(BeTrue())

	Expect(i.IsInProgressStep()).To(BeTrue(), "should successfully start an installation")
}

// IsCreateSiteStep checks if installer is at the initial step
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

// IsRequirementsReviewStep checks if installer is at the requirements step
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
	elems, err := i.page.All(".grv-slider-value-desc span").Elements()
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

// WaitForCompletion waits for ongoing install operation to be completed
func (i *Installer) WaitForCompletion() {
	Expect(i.IsInProgressStep()).To(BeTrue(), "should be in progress")
	installTimeout := defaults.InstallTimeout
	if framework.TestContext.Extensions.InstallTimeout != 0 {
		installTimeout = framework.TestContext.Extensions.InstallTimeout.Duration()
	}
	Eventually(func() bool {
		return i.IsInstallCompleted() || i.IsInstallFailed()
	}, installTimeout, defaults.InstallCompletionPollInterval).Should(BeTrue(), "wait until timeout or install success/fail message")

	Expect(i.IsInstallFailed()).To(BeFalse(), "should not fail")
	Expect(i.IsInstallCompleted()).To(BeTrue(), "should be completed")
}

func (i *Installer) proceedToReqs() {
	log.Infof("trying to init install operation")
	Expect(i.page.FindByClass("grv-installer-btn-new-site").Click()).To(Succeed())
	Eventually(func() bool {
		return i.hasIssues() || i.IsRequirementsReviewStep()
	}, defaults.InstallCreateClusterTimeout).Should(BeTrue())
	Expect(i.hasIssues()).To(BeFalse(), "should not have validation errors")
	Expect(i.IsRequirementsReviewStep()).To(BeTrue(), "should be on requirements step")
}

func (i *Installer) hasIssues() bool {
	return i.IsWarningVisible() || utils.HasValidationErrors(i.page)
}

func (i *Installer) navigateTo(URL string) {
	Expect(i.page.Navigate(URL)).To(Succeed())
	Eventually(func() bool {
		return utils.IsInstaller(i.page) || utils.IsErrorPage(i.page)
	}, defaults.AppLoadTimeout).Should(BeTrue())

	Expect(utils.IsInstaller(i.page)).To(BeTrue(), "valid installer page")
	utils.PauseForPageJs()
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

	type serverProfile struct {
		Nodes int
	}

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
	log.Infof("specifying domain name")
	Expect(page.FindByName("domainName").Fill(domainName)).To(Succeed())
	Eventually(func() bool {
		return utils.IsFound(page, ".fa-check") || utils.HasValidationErrors(page)
	}, defaults.AjaxCallTimeout).Should(BeTrue())

	Expect(page.FindByClass("fa-check")).To(BeFound(), "should be a valid cluster name")
}
