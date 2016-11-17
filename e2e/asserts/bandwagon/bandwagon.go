package bandwagon

import (
	"github.com/gravitational/robotest/e2e/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
)

func SubmitForm(page *agouti.Page, domainName string, userName string, psw string) {
	By("filling out the forms")
	band := ui.OpenBandWagon(page, domainName, userName, psw)
	band.SubmitForm()

	By("verying endpoints")
	endpoints := band.GetEndPoints()
	Expect(endpoints).NotTo(BeEmpty())
}
