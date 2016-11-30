package bandwagon

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
)

func Complete(page *web.Page, domainName string, username string, password string,
	remoteAccess, localEndpoint bool) {
	bandwagon := ui.OpenBandwagon(page, domainName, username, password)

	By("submitting bandwagon form")
	endpoints := bandwagon.SubmitForm(remoteAccess)
	Expect(len(endpoints)).To(BeNumerically(">", 0))

	if localEndpoint {
		By("using local application endpoint")
		// Use bandwagon login for both entry/service access
		login := framework.Login{Username: username, Password: password}
		serviceLogin := &framework.ServiceLogin{Username: username, Password: password}
		framework.UpdateEntry(endpoints[0], login, serviceLogin)
	}
}
