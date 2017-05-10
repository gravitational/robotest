package site

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gravitational/robotest/e2e/framework"
	ui "github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/defaults"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type Site struct {
	domainName string
	page       *web.Page
}

func Open(page *web.Page, domainName string) Site {
	site := Site{page: page, domainName: domainName}
	url := site.formatUrl("")
	VerifySiteNavigation(page, url)
	return site
}

func (s *Site) GetSiteAppPage() SiteAppPage {
	return SiteAppPage{page: s.page}
}

func (s *Site) GetSiteServerPage() SiteServerPage {
	return SiteServerPage{page: s.page}
}

func (s *Site) NavigateToSiteApp() {
	url := s.formatUrl("")
	VerifySiteNavigation(s.page, url)
}

func (s *Site) NavigateToServers() {
	url := s.formatUrl("servers")
	VerifySiteNavigation(s.page, url)

	Eventually(func() bool {
		count, _ := s.page.All(".grv-site-servers .grv-table td").Count()
		return count > 0
	}, defaults.ServerLoadTimeout).Should(
		BeTrue(),
		"waiting for servers to load")

	ui.PauseForPageJs()
}

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

	var siteEndpoints []string
	for _, v := range endpoints {
		if strings.Contains(v, strconv.Itoa(defaults.GravityHTTPPort)) {
			siteEndpoints = append(siteEndpoints, v)
		}
	}
	return siteEndpoints
}

func VerifySiteNavigation(page *web.Page, URL string) {
	Expect(page.Navigate(URL)).To(Succeed())
	Eventually(page.FindByClass("grv-site"), defaults.ElementTimeout).Should(BeFound(), "waiting for site to be ready")
	ui.PauseForComponentJs()
}

func (s *Site) formatUrl(newPrefix string) string {
	urlPrefix := fmt.Sprintf("/web/site/%v/%v", s.domainName, newPrefix)
	url, err := s.page.URL()
	Expect(err).NotTo(HaveOccurred())
	return framework.URLPathFromString(url, urlPrefix)
}
