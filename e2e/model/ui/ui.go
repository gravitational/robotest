package ui

import (
	"github.com/gravitational/robotest/e2e/model/ui/bandwagon"
	"github.com/gravitational/robotest/e2e/model/ui/installer"
	"github.com/gravitational/robotest/e2e/model/ui/opscenter"
	"github.com/gravitational/robotest/e2e/model/ui/site"
	"github.com/gravitational/robotest/e2e/model/ui/user"
	web "github.com/sclevine/agouti"
)

// UI is a facade for accessing high level ui model objects
type UI struct {
	page *web.Page
}

// Init creates UI instance
func Init(page *web.Page) UI {
	return UI{page: page}
}

// GoToSite navigates to cluster page and returns cluster object
func (u *UI) GoToSite(domainName string) site.Site {
	return site.Open(u.page, domainName)
}

// GoToInstaller navigates to installer page and returns installer object
func (u *UI) GoToInstaller(URL string) installer.Installer {
	return installer.Open(u.page, URL)
}

// GoToOpsCenter navigates to opscenter page and returns opscenter object
func (u *UI) GoToOpsCenter(URL string) opscenter.OpsCenter {
	return opscenter.Open(u.page, URL)
}

// GoToBandwagon navigates to bandwagon page and returns bandwagon object
func (u *UI) GoToBandwagon(domainName string) bandwagon.Bandwagon {
	return bandwagon.Open(u.page, domainName)
}

// EnsureUser navigates to given URL and ensures that a user is logged in
func (u *UI) EnsureUser(URL string) {
	user.EnsureUser(u.page, URL)
}
