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
	page       *web.Page
	domainName string
}

func OpenBandwagon(page *web.Page, domainName string) Bandwagon {
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
		page,
		domainName,
	}
}

func (b *Bandwagon) SubmitForm(config framework.BandwagonConfig) {
	if config.Organization == "" {
		config.Organization = defaults.BandwagonOrganization
	}

	if config.Username == "" {
		config.Username = defaults.BandwagonUsername
	}

	if config.Password == "" {
		config.Password = defaults.BandwagonPassword
	}

	if config.Email == "" {
		config.Email = defaults.BandwagonEmail
	}

	page := b.page
	By("entering email")
	log.Infof("email: %s", config.Email)
	Expect(page.FindByName("email").Fill(config.Email)).To(
		Succeed(),
		"should enter email")

	count, _ := page.FindByName("name").Count()
	if count > 0 {
		By("entering username")
		log.Infof("username: %s", config.Username)
		Expect(page.FindByName("name").Fill(config.Username)).To(
			Succeed(),
			"should enter username")
	}

	By("entering password")
	log.Infof("password: %s", config.Password)
	Expect(page.FindByName("password").Fill(config.Password)).To(
		Succeed(),
		"should enter password")

	By("re-entering password")
	Expect(page.FindByName("passwordConfirmed").Fill(config.Password)).To(
		Succeed(),
		"should re-enter password")

	count, _ = page.FindByName("org").Count()
	if count > 0 {
		By("entering organization name")
		log.Infof("organization: %s", config.Organization)
		Expect(page.FindByName("org").Fill(config.Organization)).To(
			Succeed(),
			"should enter organization name")
	}

	if config.RemoteAccess {
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
		defaults.BandwagonSubmitFormTimeout,
	).Should(BeTrue())
}
