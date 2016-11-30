package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gravitational/robotest/e2e/model/ui/defaults"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type Bandwagon struct {
	page       *web.Page
	domainName string
	email      string
	password   string
}

func OpenBandwagon(page *web.Page, domainName string, email string, password string) *Bandwagon {
	urlPrefix := fmt.Sprintf("/web/site/%v", domainName)
	r, _ := regexp.Compile("/web/.*")
	url, _ := page.URL()
	url = r.ReplaceAllString(url, urlPrefix)

	Expect(page.Navigate(url)).To(
		Succeed(),
		"should open bandwagon")

	Eventually(page.FindByClass("my-page-btn-submit"), defaults.FindTimeout).Should(
		BeFound(),
		"should wait for bandwagon to load")

	return &Bandwagon{page: page, email: email, password: password, domainName: domainName}
}

func (b *Bandwagon) SubmitForm(remoteAccess bool) {
	page := b.page

	By("entering email")
	Expect(page.FindByName("email").Fill(b.email)).To(
		Succeed(),
		"should enter email")

	By("entering password")
	Expect(page.FindByName("password").Fill(b.password)).To(
		Succeed(),
		"should enter password")

	By("re-entering password")
	Expect(page.FindByName("passwordConfirmed").Fill(b.password)).To(
		Succeed(),
		"should re-enter password")

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
}

func (b *Bandwagon) GetEndpoints() []string {
	const scriptTemplate = `
            var result = [];
            var cssSelector = ".my-page-section-endpoints-item a";
            var children = document.querySelectorAll(cssSelector);
            children.forEach( z => result.push(z.text) );
            return result; `
	var result []string

	b.page.RunScript(scriptTemplate, nil, &result)
	return result
}
