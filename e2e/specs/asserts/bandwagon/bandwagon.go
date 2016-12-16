package bandwagon

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
)

func Complete(page *web.Page, domainName string, login framework.BandwagonConfig, remoteAccess bool) (endpoints []string) {
	bandwagon := ui.OpenBandwagon(page, domainName, login.Username, login.Password, login.Organization, login.Email)

	By("submitting bandwagon form")
	endpoints = bandwagon.SubmitForm(remoteAccess)
	Expect(len(endpoints)).To(BeNumerically(">", 0))

	return endpoints
}
