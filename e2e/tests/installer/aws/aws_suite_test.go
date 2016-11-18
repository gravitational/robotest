package aws

import (
	"testing"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/ui"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
)

func TestAws(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "aws-installation")
}

var (
	driver *agouti.WebDriver
	page   *agouti.Page
)

var _ = BeforeSuite(func() {
	var err error
	driver = agouti.ChromeDriver()
	Expect(driver.Start()).To(Succeed())

	page, err = driver.NewPage()
	Expect(err).NotTo(HaveOccurred())
	ui.EnsureUser(page, framework.TestContext.StartURL,
		framework.TestContext.Login.Username,
		framework.TestContext.Login.Password,
		ui.WithGoogle)
})

var _ = AfterSuite(func() {
	Expect(driver.Stop()).To(Succeed())
})
