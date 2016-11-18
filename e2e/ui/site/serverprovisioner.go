package site

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gravitational/robotest/e2e/ui/common"

	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
	am "github.com/sclevine/agouti/matchers"
)

type ServerProvisioner struct {
	page *agouti.Page
}

type ServerProvisionerItem struct {
	PrivateIP   string `json:"PrivateIP"`
	PublicIP    string `json:"PublicIP"`
	Profile     string `json:"Profile"`
	Hostname    string `json:"Hostname"`
	InstaceType string
}

func (self *ServerProvisioner) GetServerItems() []ServerProvisionerItem {
	var items []ServerProvisionerItem
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

func (self *ServerProvisioner) AddAwsServer(
	accessKey string, secretKey string, profileLable string, instanceType string) *ServerProvisionerItem {
	page := self.page

	currentServerItems := self.GetServerItems()

	Expect(page.FindByClass("grv-site-servers-provisioner-add-new").Click()).To(
		Succeed(),
		"should click on Provision new button")

	common.FillOutAwsKeys(page, accessKey, secretKey)

	Expect(page.Find(".grv-site-servers-provisioner-content .btn-primary").Click()).To(
		Succeed(),
		"click on continue")

	Eventually(page.FindByClass("grv-site-servers-provisioner-new"), waitForElement).Should(
		am.BeFound(),
		"should display profile and instance type")

	setDropDownValue(page, "grv-site-servers-provisioner-new-profile", profileLable)
	setDropDownValue(page, "grv-site-servers-provisioner-new-instance-type", instanceType)

	Expect(page.FindByClass("grv-site-servers-btn-start").Click()).To(
		Succeed(),
		"should click on start button")

	self.expectProgressIndicator()

	common.Pause(10 * time.Second)

	updatedItems := self.GetServerItems()

	var newItem *ServerProvisionerItem

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

func (self *ServerProvisioner) DeleteAwsServer(accessKey string, secretKey string, itemToDelete *ServerProvisionerItem) {
	itemBeforeDelete := self.GetServerItems()
	self.clickDeleteServer(itemToDelete.Hostname)
	self.confirmAwsDelete(accessKey, secretKey)
	self.expectProgressIndicator()
	itemsAfterDelete := self.GetServerItems()
	Expect(len(itemsAfterDelete) < len(itemBeforeDelete)).To(
		BeTrue(),
		"very that server disappeared from the list")
}

func (self *ServerProvisioner) confirmAwsDelete(accessKey string, secretKey string) {
	common.FillOutAwsKeys(self.page, accessKey, secretKey)
	Expect(self.page.Find(".modal-dialog .btn-danger").Click()).To(
		Succeed(),
		"should click on confirmation button",
	)

}

func (self *ServerProvisioner) expectProgressIndicator() {
	page := self.page
	Eventually(page.FindByClass("grv-site-servers-operation-progress"), waitForElement).Should(
		am.BeFound(),
		"should find progress indicator")

	Eventually(page.FindByClass("grv-site-servers-operation-progress"), waitForOperation).ShouldNot(
		am.BeFound(),
		"should wait for progress indicator to disappear")
}

func (self *ServerProvisioner) clickDeleteServer(hostname string) {
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
