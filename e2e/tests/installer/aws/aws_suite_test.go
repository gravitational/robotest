package aws

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gravitational/robotest/e2e/ui"
	"github.com/sclevine/agouti"
)

func TestK8s(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "k8s-onprem")
}

var (
	driver *agouti.WebDriver
	page   *agouti.Page
)

var (
	deploymentName = "alexeyk-11-111122"
	awsAccessKey   = "AKIAJ4FJNKWG3I5M3POQ"
	awsSecretKey   = "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"
	awsRegion      = "us-west-1"
	awsKeyPair     = "ops"
	awsVpc         = "vpc-6fbb610a"
	userName       = "alex@kontsevoy.com"
	password       = "test123"
	startURL       = "https://localhost:8080/web/installer/new/gravitational.io/k8s-aws/0.45.14-138"
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
