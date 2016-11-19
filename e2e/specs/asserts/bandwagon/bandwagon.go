package bandwagon

import (
	"github.com/gravitational/robotest/e2e/ui"
	. "github.com/onsi/ginkgo"

	"github.com/sclevine/agouti"
)

func Complete(page *agouti.Page, domainName string, userName string, psw string) {
	band := ui.OpenBandwagon(page, domainName, userName, psw)
	By("submitting bandwagon form")
	band.SubmitForm()

	/*By("verying endpoints")
	endpoints := band.GetEndPoints()
	Expect(endpoints).NotTo(BeEmpty())*/
}
