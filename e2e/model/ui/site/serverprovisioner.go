package site

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gravitational/robotest/e2e/framework"
	utils "github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/agent"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type SiteServerProvisioner struct {
	page *agouti.Page
}

type SiteServer struct {
	PrivateIP   string `json:"PrivateIP"`
	PublicIP    string `json:"PublicIP"`
	Profile     string `json:"Profile"`
	Hostname    string `json:"Hostname"`
	InstaceType string
}

func (p *SiteServerProvisioner) GetSiteServers() []SiteServer {
	var items []SiteServer
	var result string
	js := ` 			
        var getter = [ ["site_servers"], serverList => {
            return serverList.map(srvMap => {                
                return {
					PublicIP: srvMap.get("public_ipv4"), 
					PrivateIP: srvMap.get("advertise_ip"),
					Hostname: srvMap.get("hostname"),                
					Profile: srvMap.get("role")					
            }
            }).toJS();
        }];

        var data = window.reactor.evaluate(getter)
        return JSON.stringify(data);	
	`
	p.page.RunScript(js, nil, &result)
	json.Unmarshal([]byte(result), &items)

	return items
}

func (p *SiteServerProvisioner) GetAgentServers() []agent.AgentServer {
	var agentServers = []agent.AgentServer{}
	s := p.page.All(".grv-provision-req-server")

	elements, _ := s.Elements()

	for index, _ := range elements {
		agentServers = append(agentServers, agent.CreateAgentServer(p.page, index))
	}

	return agentServers
}

func (p *SiteServerProvisioner) StartOnPremOperation() *SiteServer {
	currentServerItems := p.GetSiteServers()

	Expect(p.page.FindByClass("grv-site-servers-btn-start").Click()).To(
		Succeed(),
		"should start expand operation")

	//give it some time to appear on UI
	utils.Pause(10 * time.Second)

	p.expectProgressIndicator()

	updatedItems := p.GetSiteServers()

	var newItem *SiteServer

	for _, item := range updatedItems {
		for _, existedItem := range currentServerItems {
			if item.PrivateIP == existedItem.PrivateIP {
				break
			}

			newItem = &item
		}

	}

	Expect(newItem).ToNot(
		BeNil(),
		"Should find a new server in the server list")

	return newItem
}

func (p *SiteServerProvisioner) InitOnPremOperation() string {
	page := p.page

	Expect(page.FindByClass("grv-site-servers-provisioner-add-existing").Click()).To(
		Succeed(),
		"should click on Add Existing button")

	Expect(page.FindByClass("grv-control-radio-indicator").Click()).To(
		Succeed(),
		"should select first available profile")

	utils.Pause()

	Expect(page.Find(".grv-site-servers-provisioner-content .btn-primary").Click()).To(
		Succeed(),
		"should click on continue button")

	utils.Pause()

	element := page.Find(".grv-installer-server-instruction span")

	Eventually(element, defaults.AjaxCallTimeout).Should(
		BeFound(),
		"should find a command")

	command, _ := element.Text()

	Expect(command).NotTo(
		BeEmpty(),
		"command must be defined")

	return command
}

func (p *SiteServerProvisioner) AddAwsServer(
	awsConfig framework.AWSConfig, profileLable string, instanceType string) *SiteServer {
	page := p.page

	currentServerItems := p.GetSiteServers()

	Expect(page.FindByClass("grv-site-servers-provisioner-add-new").Click()).To(
		Succeed(),
		"should click on Provision new button")

	utils.Pause()

	utils.FillOutAwsKeys(page, awsConfig.AccessKey, awsConfig.SecretKey)

	Expect(page.Find(".grv-site-servers-provisioner-content .btn-primary").Click()).To(
		Succeed(),
		"click on continue")

	Eventually(page.FindByClass("grv-site-servers-provisioner-new"), defaults.ElementTimeout).Should(
		BeFound(),
		"should display profile and instance type")

	setDropDownValue(page, "grv-site-servers-provisioner-new-profile", defaults.ProfileLabel)
	setDropDownValue(page, "grv-site-servers-provisioner-new-instance-type", instanceType)

	Expect(page.FindByClass("grv-site-servers-btn-start").Click()).To(
		Succeed(),
		"should click on start button")

	p.expectProgressIndicator()

	updatedItems := p.GetSiteServers()

	var newItem *SiteServer

	for _, item := range updatedItems {
		for _, existedItem := range currentServerItems {
			if item.PrivateIP == existedItem.PrivateIP {
				break
			}

			newItem = &item
		}

	}

	Expect(newItem).ToNot(
		BeNil(),
		"Should find a new server in the server list")

	return newItem
}

func (p *SiteServerProvisioner) DeleteAwsServer(awsConfig framework.AWSConfig, itemToDelete *SiteServer) {
	p.deleteServer(itemToDelete, &awsConfig)
}

func (p *SiteServerProvisioner) DeleteOnPremServer(itemToDelete *SiteServer) {
	p.deleteServer(itemToDelete, nil)
}

func (p *SiteServerProvisioner) deleteServer(itemToDelete *SiteServer, awsConfig *framework.AWSConfig) {
	itemBeforeDelete := p.GetSiteServers()
	p.clickDeleteServer(itemToDelete.Hostname)

	if awsConfig != nil {
		utils.FillOutAwsKeys(p.page, awsConfig.AccessKey, awsConfig.SecretKey)
	}

	Expect(p.page.Find(".modal-dialog .btn-danger").Click()).To(
		Succeed(),
		"should click on confirmation button",
	)

	p.expectProgressIndicator()
	itemsAfterDelete := p.GetSiteServers()
	Expect(len(itemsAfterDelete) < len(itemBeforeDelete)).To(
		BeTrue(),
		"very that server disappeared from the list")
}

func (p *SiteServerProvisioner) expectProgressIndicator() {
	page := p.page
	Eventually(page.FindByClass("grv-site-servers-operation-progress"), defaults.ElementTimeout).Should(
		BeFound(),
		"should find progress indicator")

	Eventually(page.FindByClass("grv-site-servers-operation-progress"), defaults.OperationTimeout).ShouldNot(
		BeFound(),
		"should wait for progress indicator to disappear")

	// give some time to let all UI components to update
	utils.Pause(10 * time.Second)
}

func (p *SiteServerProvisioner) clickDeleteServer(hostname string) {
	var result int
	page := p.page

	js := ` 			
		var targetIndex = -1;
		var rows = document.querySelectorAll(".grv-site-servers .grv-table tr");
		rows.forEach( (z, index) => {
			if( z.innerText.indexOf("%v") !== -1) targetIndex = index; 
		})
		
		return targetIndex;					
	`

	js = fmt.Sprintf(js, hostname)

	page.RunScript(js, nil, &result)
	btnPath := fmt.Sprintf(".grv-site-servers .grv-table tr:nth-child(%v) .fa-trash", result)
	Expect(page.Find(btnPath).Click()).To(
		Succeed(),
		"should find and click on server delete button")

}

func setDropDownValue(page *agouti.Page, classPath string, value string) {
	if !strings.HasPrefix(classPath, ".") {
		classPath = "." + classPath
	}

	var result []string
	page.Find(classPath).Click()

	js := ` var result = []; var cssSelector = "%v .dropdown-menu a"; var children = document.querySelectorAll(cssSelector); children.forEach( z => result.push(z.innerText) ); return result; `
	js = fmt.Sprintf(js, classPath)

	page.RunScript(js, nil, &result)

	for index, optionValue := range result {
		if optionValue == value {
			optionClass := fmt.Sprintf("%v li:nth-child(%v) a", classPath, index+1)
			Expect(page.Find(optionClass).Click()).To(
				Succeed(),
				"should select given dropdown value")

			return
		}
	}

	Expect(false).To(BeTrue(), "given dropdown value does not exist")
}
