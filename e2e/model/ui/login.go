package ui

import (
	"time"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type AuthType string

const (
	WithEmail  AuthType = "email"
	WithGoogle AuthType = "google"
)

func EnsureUser(page *web.Page, URL string, username string, password string, authType AuthType) {
	Expect(page.Navigate(URL)).To(Succeed())
	count, _ := page.FindByClass("grv-user-login").Count()

	if count != 0 {
		user := CreateUser(page, username, password)
		switch authType {
		case WithEmail:
			user.LoginWithEmail()
		case WithGoogle:
			user.LoginWithGoogle()
		default:
			framework.Failf("unknown auth type %q", authType)
		}

		time.Sleep(defaults.ShortTimeout)
	}
}

func CreateUser(page *web.Page, email string, password string) User {
	return User{page: page, email: email, password: password}
}

type User struct {
	page     *web.Page
	email    string
	password string
}

func (u *User) NavigateToLogin() {
	urlS, err := u.page.URL()
	Expect(err).NotTo(HaveOccurred())
	url := URLPath(urlS, "/web/login")

	Expect(u.page.Navigate(url)).To(Succeed())
	Eventually(u.page.FindByClass("grv-user-login"), defaults.FindTimeout).Should(BeFound())
}

func (u *User) LoginWithEmail() {
	Expect(u.page.FindByName("email").Fill(u.email)).To(Succeed())
	Expect(u.page.FindByName("password").Fill(u.password)).To(Succeed())
	Expect(u.page.FindByClass("btn-primary").Click()).To(Succeed())
	Eventually(u.page.URL, defaults.FindTimeout).ShouldNot(HaveSuffix("/login"))
}

func (u *User) LoginWithGoogle() {
	Expect(u.page.FindByClass("btn-google").Click()).To(Succeed())
	Expect(u.page.FindByID("Email").Fill(u.email)).To(Succeed())
	Expect(u.page.FindByID("next").Click()).To(Succeed())
	Eventually(u.page.FindByID("Passwd"), defaults.FindTimeout).Should(BeFound())

	time.Sleep(defaults.PauseTimeout)

	Expect(u.page.FindByID("Passwd").Fill(u.password)).To(Succeed())
	Expect(u.page.FindByID("signIn").Click()).To(Succeed())

	time.Sleep(defaults.PauseTimeout)

	allowButton := u.page.FindByID("submit_approve_access")
	count, _ := allowButton.Count()

	if count > 0 {
		Expect(allowButton.Click()).To(Succeed())
	}

}

func (u *User) Signout() {
	Eventually(u.page.FindByClass("fa-sign-out"), defaults.FindTimeout).Should(BeFound())
	Expect(u.page.FindByClass("fa-sign-out").Click()).To(Succeed())
	Eventually(u.page.FindByClass("grv-user-login")).Should(BeFound())
}
