package bandwagon

import (
	"github.com/gravitational/robotest/e2e/model/ui"

	. "github.com/onsi/ginkgo"
	web "github.com/sclevine/agouti"
)

func Complete(page *web.Page, domainName string, username string, password string) {
	bandwagon := ui.OpenBandwagon(page, domainName, username, password)
	By("submitting bandwagon form")
	bandwagon.SubmitForm()

	/*By("verying endpoints")
	endpoints := band.GetEndPoints()
	Expect(endpoints).NotTo(BeEmpty())*/
}
