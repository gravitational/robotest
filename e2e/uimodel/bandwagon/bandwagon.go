/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	count, _ := b.page.FindByName("org").Count()
	if count > 0 {
		log.Infof("entering organization: %s", config.Organization)
		Expect(b.page.FindByName("org").Fill(config.Organization)).To(Succeed(), "should enter organization")
	}

	log.Infof("entering email: %s", config.Email)
	Expect(b.page.FindByName("email").Fill(config.Email)).To(Succeed(), "should enter email")
	count, _ = b.page.FindByName("name").Count()
	if count > 0 {
		log.Infof("entering username: %s", config.Username)
		Expect(b.page.FindByName("name").Fill(config.Username)).To(Succeed(), "should enter username")
	}

	log.Infof("entering password: %s", config.Password)
	Expect(b.page.FindByName("password").Fill(config.Password)).To(Succeed(), "should enter password")
	Expect(b.page.FindByName("passwordConfirmed").Fill(config.Password)).
		To(Succeed(), "should re-enter password")

	log.Info("entering extra fields")
	b.FillExtraFields(config.Extra.PlatformDNS, "platformDns")
	b.FillExtraFields(config.Extra.NFSServer, "nfsServer")
	b.FillExtraFields(config.Extra.NFSPath, "nfsPath")

	log.Info("specifying remote access")
	utils.SelectRadio(b.page, ".my-page-section .grv-control-radio", func(value string) bool {
		prefix := "Disable remote"
		if config.RemoteAccess {
			prefix = "Enable remote"
		}
		return strings.HasPrefix(value, prefix)
	})

	log.Info("submitting the form")
	Expect(b.page.FindByClass("my-page-btn-submit").Click()).To(Succeed(), "should click submit button")

	utils.PauseForPageJs()
	Eventually(func() bool {
		return utils.IsFound(b.page, ".my-page-btn-submit .fa-spin")
	}, defaults.BandwagonSubmitFormTimeout).Should(BeFalse(), "wait for progress indicator to disappear")
}

// FillExtraFields fills extra bandwagon fields
func (b *Bandwagon) FillExtraFields(fieldValue, cssSelector string) {
	count, _ := b.page.FindByName(cssSelector).Count()
	if count > 0 {
		log.Infof("entering %s: %s", cssSelector, fieldValue)
		Expect(b.page.FindByName(cssSelector).Fill(fieldValue)).To(Succeed(), "should enter %s field value", cssSelector)
	}
}
