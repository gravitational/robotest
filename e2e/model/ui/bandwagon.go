package ui

import (
	"fmt"
	"strings"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui/defaults"

	log "github.com/Sirupsen/logrus"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type Bandwagon struct {
	framework.BandwagonConfig
	page       *web.Page
	domainName string
}

func NeedsBandwagon(page *web.Page, domainName string) bool {
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
	Expect(page.RunScript(js, nil, &needsBandwagon)).To(Succeed(), "should detect if bandwagon is required")
	return needsBandwagon
}

func OpenBandwagon(page *web.Page, domainName string, config framework.BandwagonConfig) Bandwagon {
	path := fmt.Sprintf("/web/site/%v", domainName)
	url, err := page.URL()
	Expect(err).NotTo(HaveOccurred())
	url = framework.URLPathFromString(url, path)

	Expect(page.Navigate(url)).To(
		Succeed(),
		"should open bandwagon")

	Eventually(page.FindByClass("my-page-btn-submit"), defaults.FindTimeout).Should(
		BeFound(),
		"should wait for bandwagon to load")

	return Bandwagon{
		config,
		page,
		domainName,
	}
}

func (b *Bandwagon) SubmitForm(remoteAccess bool) {
	page := b.page

	By("entering email")
	log.Infof("email: %s", b.Email)
	Expect(page.FindByName("email").Fill(b.Email)).To(
		Succeed(),
		"should enter email")

	count, _ := page.FindByName("name").Count()
	if count > 0 {
		By("entering username")
		log.Infof("username: %s", b.Username)
		Expect(page.FindByName("name").Fill(b.Username)).To(
			Succeed(),
			"should enter username")
	}

	By("entering password")
	log.Infof("password: %s", b.Password)
	Expect(page.FindByName("password").Fill(b.Password)).To(
		Succeed(),
		"should enter password")

	By("re-entering password")
	Expect(page.FindByName("passwordConfirmed").Fill(b.Password)).To(
		Succeed(),
		"should re-enter password")

	count, _ = page.FindByName("org").Count()
	if count > 0 {
		By("entering organization name")
		log.Infof("organization: %s", b.Organization)
		Expect(page.FindByName("org").Fill(b.Organization)).To(
			Succeed(),
			"should enter organization name")
	}

	if remoteAccess {
		By("enabling remote access")
		SelectRadio(page, ".my-page-section .grv-control-radio", func(value string) bool {
			return strings.HasPrefix(value, "Enable remote")
		})
	} else {
		By("disabling remote access")
		SelectRadio(page, ".my-page-section .grv-control-radio", func(value string) bool {
			return strings.HasPrefix(value, "Disable remote")
		})
	}

	By("submitting the form")
	Expect(page.FindByClass("my-page-btn-submit").Click()).To(
		Succeed(),
		"should click submit button")

	PauseForPageJs()

	Eventually(
		func() bool {
			element := page.Find(".my-page-btn-submit .fa-spin")
			count, _ := element.Count()
			return count == 0
		},
		defaults.PollInterval,
	).Should(BeTrue())
}
