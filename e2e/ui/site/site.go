package site

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/lib/defaults"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type Site struct {
	page *agouti.Page
}

func Open(page *agouti.Page, domainName string) *Site {
	urlPrefix := fmt.Sprintf("/web/site/%v", framework.TestContext.ClusterName)
	r, _ := regexp.Compile("/web/.*")
	url, _ := page.URL()
	url = r.ReplaceAllString(url, urlPrefix)

	By("Navigating to installer screen")
	Expect(page.Navigate(url)).To(Succeed())
	Eventually(page.FindByClass("grv-site"), defaults.FindTimeout).Should(BeFound())

	time.Sleep(100 * time.Millisecond)

	return &Site{page: page}
}
