package site

import (
	"fmt"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sclevine/agouti"
	am "github.com/sclevine/agouti/matchers"
)

var defaultTimeout = 20 * time.Second

type Site struct {
	page *agouti.Page
}

func OpenSite(page *agouti.Page, domainName string) *Site {
	urlPrefix := fmt.Sprintf("/web/site/%v", domainName)
	r, _ := regexp.Compile("/web/.*")
	url, _ := page.URL()
	url = r.ReplaceAllString(url, urlPrefix)

	By("Navigating to installer screen")
	Expect(page.Navigate(url)).To(Succeed())
	Eventually(page.FindByClass("grv-site"), defaultTimeout).Should(am.BeFound())

	time.Sleep(100 * time.Millisecond)

	return &Site{page: page}
}
