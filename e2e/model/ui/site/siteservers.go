package site

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gravitational/robotest/e2e/framework"
	utils "github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/agent"
	"github.com/gravitational/robotest/e2e/model/ui/defaults"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type SiteServerPage struct {
	page *web.Page
}

type SiteServer struct {
	AdvertiseIP string `json:"AdvertiseIP"`
	PublicIP    string `json:"PublicIP"`
	Profile     string `json:"Profile"`
	Hostname    string `json:"Hostname"`
	InstaceType string
}

func (p *SiteServerPage) GetSiteServers() []SiteServer {
	const script = `
            var getter = [ ["site_servers"], serverList => {
                return serverList.map(srvMap => {                
                    return {
                        PublicIP: srvMap.get("public_ipv4"), 
                        AdvertiseIP: srvMap.get("advertise_ip"),
                        Hostname: srvMap.get("hostname"),
                        Profile: srvMap.get("role")
                    }
                }).toJS();
            }];

            var data = window.reactor.evaluate(getter)
            return JSON.stringify(data);
        `
	var items []SiteServer
	var result string

	p.page.RunScript(script, nil, &result)
	Expect(json.Unmarshal([]byte(result), &items)).To(Succeed())

	return items
}

func (p *SiteServerPage) GetAgentServers() []agent.AgentServer {
	var agentServers = []agent.AgentServer{}
	s := p.page.All(".grv-provision-req-server")

	elements, err := s.Elements()
	Expect(err).NotTo(HaveOccurred())

	for index, _ := range elements {
		agentServers = append(agentServers, agent.CreateAgentServer(p.page, index))
	}

	return agentServers
}

func (p *SiteServerPage) StartOnPremOperation() *SiteServer {
	currentItems := p.GetSiteServers()

	Expect(p.page.FindByClass("grv-site-servers-btn-start").Click()).To(
		Succeed(),
		"should start expand operation")

	// give it some time to appear on UI
	utils.Pause(10 * time.Second)

	p.expectProgressIndicator()

	updatedItems := p.GetSiteServers()

	var newItem *SiteServer

	for i, item := range updatedItems {
		for _, existingItem := range currentItems {
			if item.AdvertiseIP == existingItem.AdvertiseIP {
				break
			}
			newItem = &updatedItems[i]
		}
	}

	Expect(newItem).ToNot(
		BeNil(),
		"should find a new server in the server list")

	return newItem
}

func (p *SiteServerPage) InitOnPremOperation() string {
	page := p.page

	Expect(page.FindByClass("grv-site-servers-provisioner-add-existing").Click()).To(
		Succeed(),
		"should click on Add Existing button")

	utils.PauseForComponentJs()

	Expect(page.FindByClass("grv-control-radio-indicator").Click()).To(
		Succeed(),
		"should select first available profile")

	utils.PauseForComponentJs()

	Expect(page.Find(".grv-site-servers-provisioner-content .btn-primary").Click()).To(
		Succeed(),
		"should click on continue button")

	utils.PauseForComponentJs()

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

func (p *SiteServerPage) AddAWSServer(config framework.AWSConfig, profileLabel string) *SiteServer {
	page := p.page

	currentServerItems := p.GetSiteServers()

	Expect(page.FindByClass("grv-site-servers-provisioner-add-new").Click()).To(
		Succeed(),
		"should click on Provision new button")

	utils.PauseForComponentJs()

	utils.FillOutAWSKeys(page, config.AccessKey, config.SecretKey)

	Expect(page.Find(".grv-site-servers-provisioner-content .btn-primary").Click()).To(
		Succeed(),
		"click on continue")

	Eventually(page.FindByClass("grv-site-servers-provisioner-new"), defaults.ElementTimeout).Should(
		BeFound(),
		"should display profile and instance type")

	utils.SetDropdownValue2(page, "grv-site-servers-provisioner-new-profile", "", profileLabel)
	utils.SetDropdownValue2(page, "grv-site-servers-provisioner-new-instance-type", "", config.InstanceType)

	Expect(page.FindByClass("grv-site-servers-btn-start").Click()).To(
		Succeed(),
		"should click on start button")

	p.expectProgressIndicator()

	updatedItems := p.GetSiteServers()

	var newItem *SiteServer

	for i, item := range updatedItems {
		for _, existingItem := range currentServerItems {
			if item.AdvertiseIP == existingItem.AdvertiseIP {
				break
			}

			newItem = &updatedItems[i]
		}
	}

	Expect(newItem).ToNot(
		BeNil(),
		"should find a new server in the server list")

	return newItem
}

func (p *SiteServerPage) DeleteAWSServer(awsConfig framework.AWSConfig, itemToDelete *SiteServer) {
	p.deleteServer(itemToDelete, &awsConfig)
}

func (p *SiteServerPage) DeleteOnPremServer(itemToDelete *SiteServer) {
	p.deleteServer(itemToDelete, nil)
}

func (p *SiteServerPage) deleteServer(server *SiteServer, config *framework.AWSConfig) {
	p.clickDeleteServer(server.Hostname)

	if config != nil {
		utils.FillOutAWSKeys(p.page, config.AccessKey, config.SecretKey)
	}

	Expect(p.page.Find(".modal-dialog .btn-danger").Click()).To(
		Succeed(),
		"should click on confirmation button",
	)

	p.expectProgressIndicator()

	Eventually(p.serverInList(server)).ShouldNot(
		BeTrue(),
		"verify that the server disappeared from the list")
}

func (p *SiteServerPage) serverInList(server *SiteServer) func() bool {
	return func() bool {
		for _, existingServer := range p.GetSiteServers() {
			if existingServer.Hostname == server.Hostname {
				return true
			}
		}
		return false
	}
}

func (p *SiteServerPage) expectProgressIndicator() {
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

func (p *SiteServerPage) clickDeleteServer(hostname string) {
	const scriptTemplate = `
            var targetIndex = -1;
            var rows = document.querySelectorAll(".grv-site-servers .grv-table tr");
            rows.forEach( (z, index) => {
                if( z.innerText.indexOf("%v") !== -1) targetIndex = index; 
            })
            
            return targetIndex;
        `
	var result int
	page := p.page

	script := fmt.Sprintf(scriptTemplate, hostname)

	page.RunScript(script, nil, &result)
	buttonPath := fmt.Sprintf(".grv-site-servers .grv-table tr:nth-child(%v) .fa-trash", result)
	Expect(page.Find(buttonPath).Click()).To(
		Succeed(),
		"should find and click on server delete button")
}
