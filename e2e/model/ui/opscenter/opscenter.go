package opscenter

import (
	"fmt"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/defaults"

	log "github.com/Sirupsen/logrus"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type OpsCenter struct {
	page *web.Page
	url  string
}

func Open(page *web.Page, url string) OpsCenter {
	Expect(page.Navigate(url)).To(Succeed())
	Eventually(page.FindByClass("grv-portal"), defaults.ElementTimeout).Should(BeFound(), "waiting for opscenter to load")
	ui.PauseForComponentJs()
	return OpsCenter{page: page, url: url}
}

func (o *OpsCenter) DeleteSite(domainName string) {
	log.Infof("selecting a delete site item")
	deploymentIndex := getDeploymentIndex(o.page, domainName)
	Expect(deploymentIndex).To(BeNumerically(">=", 0), "expected to find a valid deployment index")
	ui.SetDropdownValue2(o.page, fmt.Sprintf(".grv-portal-sites tr:nth-child(%v)", deploymentIndex+1), "button", "Delete...")

	log.Infof("entering AWS credentials")
	elems := o.page.FindByName("aws_access_key")
	count, _ := elems.Count()
	if count > 0 {
		Expect(elems).To(BeFound(), "expected to find an input field for AWS access key")
		Expect(elems.SendKeys(framework.TestContext.AWS.AccessKey)).To(Succeed(), "expected to input AWS access key")

		elems = o.page.FindByName("aws_secret_key")
		Expect(elems).To(BeFound(), "expected to find an input field for AWS secret key")
		Expect(elems.SendKeys(framework.TestContext.AWS.SecretKey)).To(Succeed(), "expected to input AWS secret key")
	}

	log.Infof("entering domain name")
	elems = o.page.FindByName("deploymentName")
	Expect(elems).To(BeFound())
	Expect(elems.SendKeys(domainName)).To(Succeed())

	log.Infof("confirming the action")
	Expect(o.page.Find(".grv-dialog .btn-danger").Click()).To(Succeed())
	Eventually(
		func() bool {
			return getDeploymentIndex(o.page, domainName) >= 0
		}, defaults.OpsCenterDeleteSiteTimeout).ShouldNot(BeTrue(),
		fmt.Sprintf("deployment %q should disappear from the deployment list", domainName))
}

func getDeploymentIndex(page *web.Page, domainName string) int {
	const scriptTemplate = `
            var rows = Array.prototype.slice.call(document.querySelectorAll(".grv-portal-sites .grv-table .grv-portal-sites-tag"));
            return rows.findIndex( (tag) => {
		    return (tag.innerText == "Name:%v");
            });
        `

	script := fmt.Sprintf(scriptTemplate, domainName)
	var deploymentIndex int

	Expect(page.RunScript(script, nil, &deploymentIndex)).To(Succeed())

	return deploymentIndex
}
