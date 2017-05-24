package bandwagon

import (
	"fmt"
	"strings"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/uimodel/defaults"
	"github.com/gravitational/robotest/e2e/uimodel/utils"

	log "github.com/Sirupsen/logrus"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

// Bandwagon is bandwagon ui model
type Bandwagon struct {
	page *web.Page
}

// Open navigates to bandwagon URL and returns its ui model
func Open(page *web.Page, domainName string) Bandwagon {
	path := fmt.Sprintf("/web/site/%v", domainName)
	url := utils.FormatUrl(page, path)
	Expect(page.Navigate(url)).To(Succeed(), "should open bandwagon")
	Eventually(page.FindByClass("my-page-btn-submit"), defaults.FindTimeout).
		Should(BeFound(), "should wait for bandwagon to load")
	return Bandwagon{page}
}

// SubmitForm submits bandwagon form
func (b *Bandwagon) SubmitForm(config framework.BandwagonConfig) {
	log.Infof("trying to submit bandwagon form")
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

	log.Infof("entering email: %s", config.Email)
	Expect(b.page.FindByName("email").Fill(config.Email)).To(Succeed(), "should enter email")

	count, _ := b.page.FindByName("name").Count()
	if count > 0 {
		log.Infof("entering username: %s", config.Username)
		Expect(b.page.FindByName("name").Fill(config.Username)).To(Succeed(), "should enter username")
	}

	log.Infof("entering password: %s", config.Password)
	Expect(b.page.FindByName("password").
		Fill(config.Password)).To(Succeed(), "should enter password")
	Expect(b.page.FindByName("passwordConfirmed").
		Fill(config.Password)).To(Succeed(), "should re-enter password")

	log.Infof("specifying remote access")
	if config.RemoteAccess {
		utils.SelectRadio(b.page, ".my-page-section .grv-control-radio", func(value string) bool {
			return strings.HasPrefix(value, "Enable remote")
		})
	} else {
		utils.SelectRadio(b.page, ".my-page-section .grv-control-radio", func(value string) bool {
			return strings.HasPrefix(value, "Disable remote")
		})
	}

	log.Infof("submitting the form")
	Expect(b.page.FindByClass("my-page-btn-submit").Click()).To(Succeed(), "should click submit button")
	utils.PauseForPageJs()
	Eventually(
		func() bool {
			element := b.page.Find(".my-page-btn-submit .fa-spin")
			count, _ := element.Count()
			return count == 0
		},
		defaults.BandwagonSubmitFormTimeout,
	).Should(BeTrue())
}
