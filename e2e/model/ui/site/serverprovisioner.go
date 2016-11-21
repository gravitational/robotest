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

type SiteServers struct {
	page *agouti.Page
}

type SiteServerItem struct {
	PrivateIP   string `json:"PrivateIP"`
	PublicIP    string `json:"PublicIP"`
	Profile     string `json:"Profile"`
	Hostname    string `json:"Hostname"`
	InstaceType string
}

func (self *SiteServers) GetServerItems() []SiteServerItem {
	var items []SiteServerItem
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
	self.page.RunScript(js, nil, &result)
	json.Unmarshal([]byte(result), &items)

	return items
}

func (self *SiteServers) GetAgentServers() []agent.AgentServer {
	var agentServers = []agent.AgentServer{}
	s := self.page.All(".grv-provision-req-server")

	elements, _ := s.Elements()

	for index, _ := range elements {
		agentServers = append(agentServers, agent.CreateAgentServer(self.page, index))
	}

	return agentServers
}

func (self *SiteServers) StartOnPremOperation() *SiteServerItem {
	currentServerItems := self.GetServerItems()

	Expect(self.page.FindByClass("grv-site-servers-btn-start").Click()).To(
		Succeed(),
		"should start expand operation")

	utils.Pause(10 * time.Second)

	self.expectProgressIndicator()

	updatedItems := self.GetServerItems()

	var newItem *SiteServerItem

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

func (self *SiteServers) InitOnPremOperation() string {
	page := self.page

	Expect(page.FindByClass("grv-site-servers-provisioner-add-existing").Click()).To(
		Succeed(),
		"should click on Add Existing button")

	Expect(page.FindByClass("grv-control-radio-indicator").Click()).To(
		Succeed(),
		"should select first available profile")

	Expect(page.Find(".grv-site-servers-provisioner-content .btn-primary").Click()).To(
		Succeed(),
		"should click on continue button")

	element := page.Find(".grv-installer-server-instruction span")

	Expect(element).To(
		BeFound(),
		"should find a command")

	command, _ := element.Text()

	Expect(command).NotTo(
		BeEmpty(),
		"command must be defined")

	return command
}

func (self *SiteServers) AddAwsServer(
	awsConfig framework.AWSConfig, profileLable string, instanceType string) *SiteServerItem {
	page := self.page

	currentServerItems := self.GetServerItems()

	Expect(page.FindByClass("grv-site-servers-provisioner-add-new").Click()).To(
		Succeed(),
		"should click on Provision new button")

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

	self.expectProgressIndicator()

	utils.Pause(10 * time.Second)

	updatedItems := self.GetServerItems()

	var newItem *SiteServerItem

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

func (self *SiteServers) DeleteAwsServer(awsConfig framework.AWSConfig, itemToDelete *SiteServerItem) {
	itemBeforeDelete := self.GetServerItems()
	self.clickDeleteServer(itemToDelete.Hostname)
	self.confirmAwsDelete(awsConfig.AccessKey, awsConfig.SecretKey)
	self.expectProgressIndicator()
	itemsAfterDelete := self.GetServerItems()
	Expect(len(itemsAfterDelete) < len(itemBeforeDelete)).To(
		BeTrue(),
		"very that server disappeared from the list")
}

func (self *SiteServers) confirmAwsDelete(accessKey string, secretKey string) {
	utils.FillOutAwsKeys(self.page, accessKey, secretKey)
	Expect(self.page.Find(".modal-dialog .btn-danger").Click()).To(
		Succeed(),
		"should click on confirmation button",
	)

}

func (self *SiteServers) expectProgressIndicator() {
	page := self.page
	Eventually(page.FindByClass("grv-site-servers-operation-progress"), defaults.ElementTimeout).Should(
		BeFound(),
		"should find progress indicator")

	Eventually(page.FindByClass("grv-site-servers-operation-progress"), defaults.OperationTimeout).ShouldNot(
		BeFound(),
		"should wait for progress indicator to disappear")
}

func (self *SiteServers) clickDeleteServer(hostname string) {
	var result int
	page := self.page

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
