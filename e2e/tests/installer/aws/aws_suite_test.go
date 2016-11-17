package aws

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gravitational/robotest/e2e/ui"
	"github.com/sclevine/agouti"
)

func TestK8s(t *testing.T) {
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
	awsRegion      = os.Getenv("ROBO_REGION")
	awsKeyPair     = os.Getenv("ROBO_KEY_PAIRE")
	awsVpc         = os.Getenv("ROBO_VPS")
)

var _ = BeforeSuite(func() {
	var err error
	driver = agouti.ChromeDriver()
	Expect(driver.Start()).To(Succeed())

	page, err = driver.NewPage()
	Expect(err).NotTo(HaveOccurred())
	ui.EnsureUser(page, startURL, userName, password, ui.WithGoogle)
})

var _ = AfterSuite(func() {
	Expect(driver.Stop()).To(Succeed())
})
