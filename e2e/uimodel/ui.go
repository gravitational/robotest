/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package uimodel

import (
	"github.com/gravitational/robotest/e2e/uimodel/bandwagon"
	"github.com/gravitational/robotest/e2e/uimodel/installer"
	"github.com/gravitational/robotest/e2e/uimodel/opscenter"
	"github.com/gravitational/robotest/e2e/uimodel/site"
	"github.com/gravitational/robotest/e2e/uimodel/user"

	web "github.com/sclevine/agouti"
)

// UI is a facade for accessing high level ui model objects
type UI struct {
	page *web.Page
}

// InitWithUser navigates to given URL and ensures signed-in user.
func InitWithUser(page *web.Page, URL string) UI {
	ui := UI{page: page}
	user.EnsureUserAt(page, URL)
	return ui
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
func (u *UI) GoToOpsCenter() opscenter.OpsCenter {
	return opscenter.Open(u.page)
}

// GoToBandwagon navigates to bandwagon page and returns bandwagon object
func (u *UI) GoToBandwagon(domainName string) bandwagon.Bandwagon {
	return bandwagon.Open(u.page, domainName)
}
