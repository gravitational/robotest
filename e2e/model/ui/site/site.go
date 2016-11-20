package site

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
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
	By("Navigating to installer screen")
	newUrl := site.formatUrl("")
	site.assertSiteNavigation(newUrl)
	return site
}

func (s *Site) GetSiteServerProvisioner() ServerProvisioner {
	return ServerProvisioner{page: s.page}
}

func (s *Site) NavigateToServers() {
	newUrl := s.formatUrl("servers")
	s.assertSiteNavigation(newUrl)

	Eventually(func() bool {
		count, _ := s.page.All(".grv-site-servers .grv-table td").Count()
		return count > 0
	}, defaults.ServerLoadTimeout).Should(
		BeTrue(),
		"waiting for servers to load")

	ui.Pause()
}

func (s *Site) assertSiteNavigation(URL string) {
	Expect(s.page.Navigate(URL)).To(Succeed())
	Eventually(s.page.FindByClass("grv-site"), defaults.ElementTimeout).Should(BeFound(), "waiting for site to be ready")
	time.Sleep(defaults.ShortTimeout)
}

func (s *Site) formatUrl(newPrefix string) string {
	urlPrefix := fmt.Sprintf("/web/site/%v/%v", s.domainName, newPrefix)
	r, _ := regexp.Compile("/web/.*")
	url, _ := s.page.URL()
	return r.ReplaceAllString(url, urlPrefix)
}
