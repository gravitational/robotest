package login_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
)

func TestLogin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Login Suite")
}

var (
	page   *agouti.Page
	driver *agouti.WebDriver

	userName = os.Getenv("ROBO_USER_NAME")
	password = os.Getenv("ROBO_USER_PASSWORD")
	baseURL  = os.Getenv("ROBO_ENTRY_URL")
)

var _ = BeforeSuite(func() {
	var err error
	driver = agouti.ChromeDriver()
	Expect(driver.Start()).To(Succeed())

	page, err = driver.NewPage()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(driver.Stop()).To(Succeed())
})
