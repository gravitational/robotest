package user

import (
	"time"

	"github.com/gravitational/log"
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/uimodel/defaults"
	"github.com/gravitational/robotest/e2e/uimodel/utils"
	"github.com/gravitational/trace"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

const (
	WithEmail         = "email"
	WithGoogle        = "google"
	WithNoProvider    = ""
	googlePageTimeout = 1 * time.Second
)

// User contains ui user information
type User struct {
	page     *web.Page
	email    string
	password string
}

// CreateUser returns an instance of User
func CreateUser(page *web.Page, email string, password string) User {
	return User{page: page, email: email, password: password}
}

// LoginWithEmail logs in a user with email and password
func (u *User) LoginWithEmail() {
	Expect(u.page.FindByName("userId").Fill(u.email)).To(Succeed())
	Expect(u.page.FindByName("password").Fill(u.password)).To(Succeed())
	Expect(u.page.FindByClass("btn-primary").Click()).To(Succeed())
	Eventually(u.page.URL, defaults.FindTimeout).ShouldNot(HaveSuffix("/login"))
}

// LoginWithGoogle logs in a user with google
func (u *User) LoginWithGoogle() {
	const cssUserSelector = `[data-email="robotest@gravitational.io"]`
	Expect(u.page.FindByClass("btn-google").Click()).To(Succeed())

	// Check if previously has been signed-in to handle list of suggested users
	googleUsers := u.page.Find(cssUserSelector)
	userCount, _ := googleUsers.Count()
	if userCount == 0 {
		Expect(u.page.FindByID("identifierId").Fill(u.email)).To(Succeed())
		Expect(u.page.FindByID("identifierNext").Click()).To(Succeed())
		Eventually(u.page.FindByName("password"), defaults.LoginGoogleNextStepTimeout).Should(BeFound())
		utils.PauseForPageJs()
		Expect(u.page.FindByName("password").Fill(u.password)).To(Succeed())
		Expect(u.page.FindByID("passwordNext").Click()).To(Succeed())
	} else {
		Expect(u.page.Find(cssUserSelector).Click()).To(Succeed())
	}

	utils.PauseForPageJs()

	// check if google approve access page is shown
	// (this page may not appear based on the browser history)
	allowButton := u.page.FindByID("submit_approve_access")
	count, _ := allowButton.Count()
	if count > 0 {
		Eventually(
			u.page.Find("#submit_approve_access:not(:disabled)"), defaults.FindTimeout).Should(
			BeFound(),
			"should wait until google access approve button becomes active")

		Expect(allowButton.Click()).To(Succeed())
		utils.PauseForPageJs()
	}
}

// Signout logs out a user
func (u *User) Signout() {
	Eventually(u.page.FindByClass("fa-sign-out"), defaults.FindTimeout).Should(BeFound())
	Expect(u.page.FindByClass("fa-sign-out").Click()).To(Succeed())
	Eventually(u.page.FindByClass("grv-user-login")).Should(BeFound())
}

// EnsureUserAt navigates to given URL and ensures that a user is logged in
func EnsureUserAt(page *web.Page, URL string) {
	log.Infof("ensuring a logged in user at %s", URL)
	Expect(page.Navigate(URL)).To(Succeed())
	Eventually(page.FirstByClass("grv"), defaults.AppLoadTimeout).Should(BeFound())
	if utils.IsFound(page, ".grv-user-login") {
		log.Infof("handling login")
		login := framework.TestContext.Login
		user := CreateUser(page, login.Username, login.Password)
		switch login.AuthProvider {
		case WithEmail, WithNoProvider:
			user.LoginWithEmail()
		case WithGoogle:
			user.LoginWithGoogle()
		default:
			trace.BadParameter("unknown auth type %s", login.AuthProvider)
		}

		utils.PauseForComponentJs()
		Expect(page.Navigate(URL)).To(Succeed())
	}
}
