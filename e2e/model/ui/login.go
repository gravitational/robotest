package ui

import (
	"time"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui/constants"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type AuthType string

const (
	WithEmail         = "email"
	WithGoogle        = "google"
	WithNoProvider    = ""
	googlePageTimeout = 1 * time.Second
)

func EnsureUser(page *web.Page, URL string, login framework.Login) {
	Expect(page.Navigate(URL)).To(Succeed())
	count, _ := page.FindByClass("grv-user-login").Count()

	if count != 0 {
		user := CreateUser(page, login.Username, login.Password)
		switch login.AuthProvider {
		case WithNoProvider:
		case WithEmail:
			user.LoginWithEmail()
		case WithGoogle:
			user.LoginWithGoogle()
		default:
			framework.Failf("unknown auth type %q", login.AuthProvider)
		}

		PauseForComponentJs()
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
	Eventually(u.page.FindByClass("grv-user-login"), constants.FindTimeout).Should(BeFound())
}

func (u *User) LoginWithEmail() {
	Expect(u.page.FindByName("email").Fill(u.email)).To(Succeed())
	Expect(u.page.FindByName("password").Fill(u.password)).To(Succeed())
	Expect(u.page.FindByClass("btn-primary").Click()).To(Succeed())
	Eventually(u.page.URL, constants.FindTimeout).ShouldNot(HaveSuffix("/login"))
}

func (u *User) LoginWithGoogle() {
	Expect(u.page.FindByClass("btn-google").Click()).To(Succeed())
	Expect(u.page.FindByID("Email").Fill(u.email)).To(Succeed())
	Expect(u.page.FindByID("next").Click()).To(Succeed())
	Eventually(u.page.FindByID("Passwd"), constants.FindTimeout).Should(BeFound())

	PauseForPageJs()

	Expect(u.page.FindByID("Passwd").Fill(u.password)).To(Succeed())
	Expect(u.page.FindByID("signIn").Click()).To(Succeed())

	PauseForPageJs()

	allowButton := u.page.FindByID("submit_approve_access")
	count, _ := allowButton.Count()

	if count > 0 {
		Expect(allowButton.Click()).To(Succeed())
	}

}

func (u *User) Signout() {
	Eventually(u.page.FindByClass("fa-sign-out"), constants.FindTimeout).Should(BeFound())
	Expect(u.page.FindByClass("fa-sign-out").Click()).To(Succeed())
	Eventually(u.page.FindByClass("grv-user-login")).Should(BeFound())
}
