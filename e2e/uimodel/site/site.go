package site

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/uimodel/defaults"
	"github.com/gravitational/robotest/e2e/uimodel/user"
	"github.com/gravitational/robotest/e2e/uimodel/utils"
	"github.com/gravitational/trace"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
	log "github.com/sirupsen/logrus"
)

// Site is cluster ui model
type Site struct {
	domainName string
	page       *web.Page
}

// Open navigates to cluster URL and returns its ui model
func Open(page *web.Page, domainName string) Site {
	site := Site{page: page, domainName: domainName}
	url := utils.GetSiteURL(page, domainName)
	VerifySiteNavigation(page, url)
	return site
}

// GoToIndex navigates to cluster index page
func (s *Site) GoToIndex() AppPage {
	url := utils.GetSiteURL(s.page, s.domainName)
	VerifySiteNavigation(s.page, url)
	return AppPage{site: s}
}

// GoToServers navigates to cluster server page
func (s *Site) GoToServers() ServerPage {
	url := utils.GetSiteServersURL(s.page, s.domainName)
	VerifySiteNavigation(s.page, url)
	Eventually(func() bool {
		count, _ := s.page.All(".grv-site-servers .grv-table td").Count()
		return count > 0
	}, defaults.AjaxCallTimeout).Should(BeTrue(), "waiting for servers to load")

	utils.PauseForPageJs()
	return ServerPage{site: s}
}

// UpdateWithLatestVersion updates this cluster with the new version
func (s *Site) UpdateWithLatestVersion() {
	log.Infof("looking for available versions")
	appPage := s.GoToIndex()
	allNewVersions := appPage.GetNewVersions()
	Expect(allNewVersions).NotTo(BeEmpty(), "should have at least 1 new version available")

	log.Infof("starting an update operation")
	newVersion := allNewVersions[len(allNewVersions)-1]
	appPage.StartUpdateOperation(newVersion)
	Eventually(func() bool {
		log.Infof("checking if update operation has been completed")
		// Check for login since update operation may cause a logout
		if utils.IsLoginPage(s.page) || s.IsReady() {
			return true
		}

		err := s.page.Refresh()
		if err != nil {
			log.Debug(trace.DebugReport(err))
		}

		return false

	}, defaults.SiteLogoutAfterUpdateTimeout, defaults.SiteLogoutAfterUpdatePollInterval).
		Should(BeTrue(), "update operation didn't finish in timely fashion")

	log.Infof("checking if app version has been updated correctly")
	siteURL := framework.SiteURL()
	user.EnsureUserAt(s.page, siteURL)
	appPage = s.GoToIndex()
	curVer := appPage.GetCurrentVersion()
	Expect(curVer.Version).To(BeEquivalentTo(newVersion.Version), "should display the new version")
}

// GetEndpoints returns cluster endpoints
func (s *Site) GetEndpoints() (endpoints []string) {
	const scriptTemplate = `
		var urls = [];
		var endpoints = window.reactor.evaluate(['site_current', 'endpoints']);
		if(!endpoints){
			return urls;
		}

		endpoints = endpoints.toJS();
		for( var i = 0; i < endpoints.length; i ++){
			var addressess = endpoints[i].addresses || []
			addressess.forEach( a => urls.push(a))
		}            
		return urls; `

	Expect(s.page.RunScript(scriptTemplate, nil, &endpoints)).To(Succeed())
	return endpoints
}

// WaitForOperationCompletion waits for cluster ongoing operation to be completed
func (s *Site) WaitForOperationCompletion() {
	log.Infof("wait for operation completion")
	s.WaitForBusyState()
	s.WaitForReadyState()
}

// WaitForReadyState waits until cluster is ready
func (s *Site) WaitForReadyState() {
	Eventually(s.IsReady, defaults.SiteOperationEndTimeout).
		Should(BeTrue(), "should wait for progress indicator to disappear")
}

// WaitForBusyState waits until cluster is busy
func (s *Site) WaitForBusyState() {
	Eventually(s.IsBusy, defaults.SiteOperationStartTimeout).Should(BeTrue(), "should find progress indicator")
}

// IsReady checks if cluster is in ready state
func (s *Site) IsReady() bool {
	log.Infof("checking for cluster ready state")
	selection := s.page.Find(".grv-site-nav-top-indicator.--ready")
	count, _ := selection.Count()
	return count > 0
}

// IsBusy checks if cluster is in busy state
func (s *Site) IsBusy() bool {
	log.Infof("checking for cluster busy state")
	selection := s.page.Find(".grv-site-nav-top-indicator.--processing")
	count, _ := selection.Count()
	return count > 0
}

// VerifySiteNavigation navigates to given URL and ensures it is a cluster page
func VerifySiteNavigation(page *web.Page, URL string) {
	Expect(page.Navigate(URL)).To(Succeed())
	Eventually(page.FindByClass("grv-site"), defaults.AppLoadTimeout).
		Should(BeFound(), "waiting for site to be ready")
	utils.PauseForComponentJs()
}
