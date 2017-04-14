package ui

import (
	"fmt"
	"strconv"
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

func (b *Bandwagon) SubmitForm(remoteAccess bool) (endpoints []string) {
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

	Eventually(page.FindByClass("my-page-section-endpoints"), defaults.FindTimeout).Should(
		BeFound(),
		"should find endpoints")

	endpoints = b.GetEndpoints()
	log.Infof("endpoints: %q", endpoints)
	Expect(len(endpoints)).To(BeNumerically(">", 0), "expected at a single application endpoint")

	return endpoints
}

func (b *Bandwagon) GetEndpoints() (endpoints []string) {
	const scriptTemplate = `
            var endpoints = [];
            var cssSelector = ".my-page-section-endpoints-item a";
            var children = document.querySelectorAll(cssSelector);
            children.forEach(endpoint => endpoints.push(endpoint.text));
            return endpoints; `

	Expect(b.page.RunScript(scriptTemplate, nil, &endpoints)).To(Succeed())

	var siteEndpoints []string
	for _, v := range endpoints {
		if strings.Contains(v, strconv.Itoa(defaults.GravityHTTPPort)) {
			siteEndpoints = append(siteEndpoints, v)
		}
	}
	return siteEndpoints
}
