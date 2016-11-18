package site

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/ui"
	"github.com/sclevine/agouti"
)

func TestSite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "aws-installation")
}

var (
	driver *agouti.WebDriver
	page   *agouti.Page
)

var (
	deploymentName = framework.TestContext.ClusterName
	userName       = framework.TestContext.Login.Username
	password       = framework.TestContext.Login.Password
	startURL       = framework.TestContext.StartURL
	awsConfig      = framework.TestContext.AWS
	profileLabel   = "worker node"
	instanceType   = "m3.large"
)

var _ = BeforeSuite(func() {
	var err error
	driver = agouti.ChromeDriver()
	Expect(driver.Start()).To(Succeed())
	page, err = driver.NewPage()

	Expect(err).NotTo(HaveOccurred())

	ui.EnsureUser(page, startURL, userName, password, ui.WithEmail)
})

var _ = AfterSuite(func() {
	Expect(driver.Stop()).To(Succeed())
})
