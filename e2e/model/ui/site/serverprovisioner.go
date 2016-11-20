package site

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type ServerProvisioner struct {
	page *web.Page
}

type ServerProvisionerItem struct {
	PrivateIP   string `json:"PrivateIP"`
	PublicIP    string `json:"PublicIP"`
	Profile     string `json:"Profile"`
	Hostname    string `json:"Hostname"`
	InstaceType string
}

func (r *ServerProvisioner) GetServerItems() []ServerProvisionerItem {
	const scriptTemplate = ` 			
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
	var items []ServerProvisionerItem
	var result string

	Expect(r.page.RunScript(scriptTemplate, nil, &result)).To(Succeed())
	Expect(json.Unmarshal([]byte(result), &items)).To(Succeed())

	return items
}

func (r *ServerProvisioner) AddAwsServer(
	config framework.AWSConfig, profileLable string, instanceType string) ServerProvisionerItem {
	currentServerItems := r.GetServerItems()

	Expect(r.page.FindByClass("grv-site-servers-provisioner-add-new").Click()).To(
		Succeed(),
		"should click on Provision new button")

	ui.FillOutAwsKeys(r.page, config.AccessKey, config.SecretKey)

	Expect(r.page.Find(".grv-site-servers-provisioner-content .btn-primary").Click()).To(
		Succeed(),
		"click on continue")

	Eventually(r.page.FindByClass("grv-site-servers-provisioner-new"), defaults.ElementTimeout).Should(
		BeFound(),
		"should display profile and instance type")

	setDropDownValue(r.page, "grv-site-servers-provisioner-new-profile", defaults.ProfileLabel)
	setDropDownValue(r.page, "grv-site-servers-provisioner-new-instance-type", defaults.InstanceType)

	Expect(r.page.FindByClass("grv-site-servers-btn-start").Click()).To(
		Succeed(),
		"should click on start button")

	r.expectProgressIndicator()

	// FIXME: why is this not Eventually?
	ui.Pause(10 * time.Second)

	updatedItems := r.GetServerItems()

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

	return *newItem
}

func (r *ServerProvisioner) DeleteAwsServer(config framework.AWSConfig, itemToDelete ServerProvisionerItem) {
	itemBeforeDelete := r.GetServerItems()

	r.clickDeleteServer(itemToDelete.Hostname)
	r.confirmAwsDelete(config.AccessKey, config.SecretKey)
	r.expectProgressIndicator()

	itemsAfterDelete := r.GetServerItems()
	Expect(len(itemsAfterDelete) < len(itemBeforeDelete)).To(
		BeTrue(),
		"very that server disappeared from the list")
}

func (r *ServerProvisioner) confirmAwsDelete(accessKey string, secretKey string) {
	ui.FillOutAwsKeys(r.page, accessKey, secretKey)
	Expect(r.page.Find(".modal-dialog .btn-danger").Click()).To(
		Succeed(),
		"should click on confirmation button",
	)
}

func (r *ServerProvisioner) expectProgressIndicator() {
	Eventually(r.page.FindByClass("grv-site-servers-operation-progress"), defaults.ElementTimeout).Should(
		BeFound(),
		"should find progress indicator")

	Eventually(r.page.FindByClass("grv-site-servers-operation-progress"), defaults.OperationTimeout).ShouldNot(
		BeFound(),
		"should wait for progress indicator to disappear")
}

func (r *ServerProvisioner) clickDeleteServer(hostname string) {
	const scriptTemplate = `
		var targetIndex = -1;
		var rows = document.querySelectorAll(".grv-site-servers .grv-table tr");
		rows.forEach( (z, index) => {
			if( z.innerText.indexOf("%v") !== -1) targetIndex = index; 
		})
		
		return targetIndex;					
	`
	var result int

	script := fmt.Sprintf(scriptTemplate, hostname)

	r.page.RunScript(script, nil, &result)
	btnPath := fmt.Sprintf(".grv-site-servers .grv-table tr:nth-child(%v) .fa-trash", result)
	Expect(r.page.Find(btnPath).Click()).To(
		Succeed(),
		"should find and click on server delete button")
}

func setDropDownValue(page *web.Page, classPath string, value string) {
	const scriptTemplate = ` var result = []; var cssSelector = "%v .dropdown-menu a"; var children = document.querySelectorAll(cssSelector); children.forEach( z => result.push(z.innerText) ); return result; `

	if !strings.HasPrefix(classPath, ".") {
		classPath = "." + classPath
	}

	Expect(page.Find(classPath).Click()).To(Succeed())

	script := fmt.Sprintf(scriptTemplate, classPath)

	var result []string
	Expect(page.RunScript(script, nil, &result)).To(Succeed())

	for i, optionValue := range result {
		if optionValue == value {
			optionClass := fmt.Sprintf("%v li:nth-child(%v) a", classPath, i+1)
			Expect(page.Find(optionClass).Click()).To(
				Succeed(),
				"should select given dropdown value")

			return
		}
	}

	framework.Failf("failed to find value %q in dropdown", value)
}
