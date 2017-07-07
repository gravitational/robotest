package bandwagon

import (
	"strings"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/uimodel/defaults"
	"github.com/gravitational/robotest/e2e/uimodel/utils"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
	log "github.com/sirupsen/logrus"
)

// Bandwagon is bandwagon ui model
type Bandwagon struct {
	page *web.Page
}

// Open navigates to bandwagon URL and returns its ui model
func Open(page *web.Page, domainName string) Bandwagon {
	url := utils.GetSiteURL(page, domainName)
	Expect(page.Navigate(url)).To(Succeed(), "navigating to bandwagon")
	Eventually(page.FindByClass("my-page-btn-submit"), defaults.AppLoadTimeout).
		Should(BeFound(), "should wait for bandwagon to load")
	return Bandwagon{page}
}

// SubmitForm submits bandwagon form
func (b *Bandwagon) SubmitForm(config framework.BandwagonConfig) {
	log.Infof("trying to submit bandwagon form")
	log.Infof("entering email: %s", config.Email)
	Expect(b.page.FindByName("email").Fill(config.Email)).To(Succeed(), "should enter email")
	count, _ := b.page.FindByName("name").Count()
	if count > 0 {
		log.Infof("entering username: %s", config.Username)
		Expect(b.page.FindByName("name").Fill(config.Username)).To(Succeed(), "should enter username")
	}

	log.Infof("entering password: %s", config.Password)
	Expect(b.page.FindByName("password").Fill(config.Password)).To(Succeed(), "should enter password")
	Expect(b.page.FindByName("passwordConfirmed").Fill(config.Password)).
		To(Succeed(), "should re-enter password")

	log.Infof("specifying remote access")
	utils.SelectRadio(b.page, ".my-page-section .grv-control-radio", func(value string) bool {
		prefix := "Disable remote"
		if config.RemoteAccess {
			prefix = "Enable remote"
		}
		return strings.HasPrefix(value, prefix)
	})

	log.Infof("submitting the form")
	Expect(b.page.FindByClass("my-page-btn-submit").Click()).To(Succeed(), "should click submit button")

	utils.PauseForPageJs()
	Eventually(func() bool {
		return utils.IsFound(b.page, ".my-page-btn-submit .fa-spin")
	}, defaults.BandwagonSubmitFormTimeout).Should(BeFalse(), "wait for progress indicator to disappear")
}
