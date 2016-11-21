package site

import (
	"fmt"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gravitational/robotest/e2e/ui/common"
	"github.com/sclevine/agouti"
	am "github.com/sclevine/agouti/matchers"
)

var (
	waitForServersToLoad = 20 * time.Second
	waitForElement       = 20 * time.Second
	waitForOperation     = 5 * time.Minute
)

type Site struct {
	domainName string
	page       *agouti.Page
}

func Open(page *agouti.Page, domainName string) *Site {
	site := Site{page: page, domainName: domainName}
	By("Navigating to installer screen")
	newUrl := site.formatUrl("")
	site.assertSiteNavigation(newUrl)
	return &site
}

func (s *Site) GetSiteServerProvisioner() *SiteServers {
	return &SiteServers{page: s.page}
}

func (s *Site) NavigateToServers() {
	newUrl := s.formatUrl("servers")
	s.assertSiteNavigation(newUrl)

	Eventually(func() bool {
		count, _ := s.page.All(".grv-site-servers .grv-table td").Count()
		return count > 0
	}, waitForServersToLoad).Should(
		BeTrue(),
		"waiting for servers to load")

	common.Pause()
}

func (s *Site) assertSiteNavigation(URL string) {
	Expect(s.page.Navigate(URL)).To(Succeed())
	Eventually(s.page.FindByClass("grv-site"), waitForElement).Should(am.BeFound(), "waiting for site to be ready")
	time.Sleep(100 * time.Millisecond)
}

func (s *Site) formatUrl(newPrefix string) string {
	urlPrefix := fmt.Sprintf("/web/site/%v/%v", s.domainName, newPrefix)
	r, _ := regexp.Compile("/web/.*")
	url, _ := s.page.URL()
	return r.ReplaceAllString(url, urlPrefix)
}
