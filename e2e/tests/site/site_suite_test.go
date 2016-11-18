package site

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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
	deploymentName = os.Getenv("ROBO_DEPLOYMANT_NAME")
	awsAccessKey   = os.Getenv("ROBO_ACCESS_KEY")
	awsSecretKey   = os.Getenv("ROBO_SECRET_KEY")
	userName       = os.Getenv("ROBO_USER_NAME")
	password       = os.Getenv("ROBO_USER_PASSWORD")
	startURL       = os.Getenv("ROBO_ENTRY_URL")
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
