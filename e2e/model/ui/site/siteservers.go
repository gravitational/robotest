package site

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gravitational/robotest/e2e/framework"
	utils "github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/agent"
	"github.com/gravitational/robotest/e2e/model/ui/defaults"

	. "github.com/onsi/gomega"
	. "github.com/sclevine/agouti/matchers"
)

type SiteServerPage struct {
	site *Site
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

	p.site.page.RunScript(script, nil, &result)
	Expect(json.Unmarshal([]byte(result), &items)).To(Succeed())

	return items
}

func (p *SiteServerPage) GetAgentServers() []agent.AgentServer {
	var agentServers = []agent.AgentServer{}
	s := p.site.page.All(".grv-provision-req-server")

	elements, err := s.Elements()
	Expect(err).NotTo(HaveOccurred())

	for index, _ := range elements {
		cssAgentServerSelector := fmt.Sprintf(".grv-provision-req-server:nth-child(%v)", index+1)
		agentServers = append(agentServers, agent.CreateAgentServer(p.site.page, cssAgentServerSelector))
	}

	return agentServers
}

func (p *SiteServerPage) startOnPremAddServerOperation() SiteServer {
	currentItems := p.GetSiteServers()

	Expect(p.site.page.FindByClass("grv-site-servers-btn-start").Click()).To(
		Succeed(),
		"should start expand operation")

	p.waitForOperationCompletion()

	updatedItems := p.GetSiteServers()

	var newItem SiteServer

	for i, item := range updatedItems {
		for _, existingItem := range currentItems {
			if item.AdvertiseIP == existingItem.AdvertiseIP {
				break
			}
			newItem = updatedItems[i]
		}
	}

	Expect(newItem).ToNot(
		BeNil(),
		"should find a new server in the server list")

	return newItem
}

func (p *SiteServerPage) AddOnPremServer() SiteServer {
	page := p.site.page
	config := framework.TestContext.Onprem
	Expect(page.FindByClass("grv-site-servers-provisioner-add-existing").Click()).To(Succeed(), "should click on Add Existing button")

	utils.PauseForComponentJs()
	p.selectOnPremProfile(config.ExpandProfile)
	utils.PauseForComponentJs()

	Expect(page.Find(".grv-site-servers-provisioner-content .btn-primary").Click()).To(Succeed(), "should click on continue button")
	utils.PauseForComponentJs()

	element := page.Find(".grv-installer-server-instruction span")
	Eventually(element, defaults.AjaxCallTimeout).Should(BeFound(), "should find a command")

	agentCommand, _ := element.Text()
	Expect(agentCommand).NotTo(BeEmpty(), "command must be defined")

	nodes, err := framework.Cluster.Provisioner().NodePool().Allocate(1)
	Expect(err).NotTo(HaveOccurred(), "should allocate a new node")
	framework.RunAgentCommand(agentCommand, nodes[0])

	Eventually(p.GetAgentServers, defaults.AgentServerTimeout).Should(HaveLen(1), "should wait for the agent server")
	provisioner := framework.Cluster.Provisioner()
	ctx := framework.TestContext
	// TODO: store private IPs for terraform in state
	// to avoid this check
	if ctx.Provisioner != "terraform" {
		agentServers := p.GetAgentServers()
		for _, s := range agentServers {
			s.SetIPByInfra(provisioner)
		}
	}

	newServer := p.startOnPremAddServerOperation()
	Expect(newServer).NotTo(BeNil(), "new server should appear in the server table")
	return newServer
}

func (p *SiteServerPage) AddAWSServer() SiteServer {
	config := framework.TestContext.AWS
	page := p.site.page

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

	profileLabel := p.getProfileLabel(config.ExpandProfile)
	utils.SetDropdownValue2(page, ".grv-site-servers-provisioner-new-profile", "", profileLabel)

	instanceType := config.ExpandAWSInstanceType
	if instanceType == "" {
		instanceType = p.getFirstAvailableAWSInstanceType()
	}

	utils.SetDropdownValue2(page, ".grv-site-servers-provisioner-new-instance-type", "", instanceType)

	Expect(page.FindByClass("grv-site-servers-btn-start").Click()).To(
		Succeed(),
		"should click on start button")

	p.waitForOperationCompletion()

	updatedItems := p.GetSiteServers()
	var newItem SiteServer
	for i, item := range updatedItems {
		for _, existingItem := range currentServerItems {
			if item.AdvertiseIP == existingItem.AdvertiseIP {
				break
			}

			newItem = updatedItems[i]
		}
	}

	Expect(newItem).ToNot(
		BeNil(),
		"should find a new server in the server list")

	return newItem
}

func (p *SiteServerPage) DeleteServer(server SiteServer) {
	p.clickDeleteServer(server.AdvertiseIP)
	Expect(p.site.page.Find(".modal-dialog .btn-danger").Click()).To(
		Succeed(),
		"should click on confirmation button",
	)

	p.waitForOperationCompletion()
	Eventually(p.hasServer(server), defaults.SiteServerListRefreshAfterShrinkTimeout).ShouldNot(
		BeTrue(),
		"verify that the server disappeared from the list")
}

func (p *SiteServerPage) hasServer(server SiteServer) func() bool {
	return func() bool {
		for _, existingServer := range p.GetSiteServers() {
			if existingServer.Hostname == server.Hostname {
				return true
			}
		}
		return false
	}
}

func (p *SiteServerPage) waitForOperationCompletion() {
	page := p.site.page
	Eventually(page.Find(".grv-site-nav-top-indicator.--processing"), defaults.ElementTimeout).Should(
		BeFound(),
		"should find progress indicator")

	Eventually(page.Find(".grv-site-nav-top-indicator.--ready"), defaults.SiteOperationTimeout).Should(
		BeFound(),
		"should wait for progress indicator to disappear")

	utils.PauseForServerListRefresh()
}

func (p *SiteServerPage) clickDeleteServer(serverId string) {
	const scriptTemplate = `
            var targetIndex = -1;
            var rows = document.querySelectorAll(".grv-site-servers .grv-table .dropdown-toggle");
            rows.forEach( (z, index) => {
                if( z.innerText.indexOf("%v") !== -1) targetIndex = index; 
            })
            
            return targetIndex;
        `
	var result int
	page := p.site.page

	script := fmt.Sprintf(scriptTemplate, serverId)
	page.RunScript(script, nil, &result)

	result = result + 1
	actionsMenuPath := fmt.Sprintf(".grv-site-servers tr:nth-child(%v) .dropdown-toggle", result)
	Expect(page.Find(actionsMenuPath).Click()).To(
		Succeed(),
		"should find and expand action menu")

	deleteActionBtnPath := fmt.Sprintf(".grv-site-servers tr:nth-child(%v) .dropdown-menu .fa-trash", result)
	Expect(page.Find(deleteActionBtnPath).Click()).To(
		Succeed(),
		"should find and click on server delete action")
}

func (p *SiteServerPage) getFirstAvailableAWSInstanceType() string {
	var instanceType string
	const js = `
		var cssSelector = ".grv-site-servers-provisioner-new-instance-type li a"; 
		var items = document.querySelectorAll(cssSelector)			
		return items.length > 0 ? items[0].text : "";  			
	`

	Expect(p.site.page.RunScript(js, nil, &instanceType)).To(
		Succeed(),
		"should retrieve first available instance type")

	Expect(instanceType).NotTo(BeEmpty(), "cannot find any instance types")

	return instanceType
}

func (p *SiteServerPage) selectOnPremProfile(profileName string) {
	if profileName == "" {
		Expect(p.site.page.FindByClass("grv-control-radio-indicator").Click()).To(
			Succeed(),
			"should select first available profile")
		return
	}

	profileLabel := p.getProfileLabel(profileName)
	utils.SelectRadio(p.site.page, ".grv-control-radio", func(value string) bool {
		return strings.HasPrefix(value, profileLabel)
	})
}

func (p *SiteServerPage) getProfileLabel(profileName string) string {
	var profileLabel string
	siteName := p.site.domainName
	const jsTemplate = `
		var profileName = "%v";
		var server = null;		
		var nodeProfilers = reactor.evaluate(["sites", "%v", "app", "manifest", "nodeProfiles"])
			.toJS()
			.reduce( (r, item) => { r[item.name] = item; return r;}, {});
			
		if(profileName !== ""){
			server = nodeProfilers[profileName];								
		}else{
			server = nodeProfilers[0];
		}

		if(!server){
			return "";
		}

		return server.description || server.serviceRole
	`

	js := fmt.Sprintf(jsTemplate, profileName, siteName)

	Expect(p.site.page.RunScript(js, nil, &profileLabel)).To(
		Succeed(),
		fmt.Sprintf("should run js script to retrieve a label for %v profile", profileName))

	Expect(profileLabel).NotTo(
		BeEmpty(),
		fmt.Sprintf("label should not be empty for %v profile", profileName))

	return profileLabel
}
