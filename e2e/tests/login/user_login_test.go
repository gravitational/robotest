package login_test

import (
	"time"

	"github.com/gravitational/robotest/e2e/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	am "github.com/sclevine/agouti/matchers"
)

var _ = Describe("Login", func() {

	defaultTimeout := 10 * time.Second

	AssertRedirectToLogin := func() {
		By("redirecting the user to the login form from the home page", func() {
			Expect(page.Navigate(baseURL)).To(Succeed())
			Expect(page.URL()).To(HaveSuffix("web/login"))
		})
	}

	It("should manage user authentication using Google credentials", func() {
		AssertRedirectToLogin()

		By("allowing a user to use google account")
		user := ui.CreateUser(page, userName, password)
		user.LoginWithGoogle()

		Eventually(page.URL, defaultTimeout).Should(HaveSuffix("/portal"), "Waiting for portal page to load")

		By("loging out")
		Eventually(page.FindByClass("fa-sign-out"), defaultTimeout).Should(am.BeFound())
		Expect(page.FindByClass("fa-sign-out").Click()).To(Succeed())
		Eventually(page.URL, defaultTimeout).Should(HaveSuffix("/web/login"))

		AssertRedirectToLogin()
	})
})
