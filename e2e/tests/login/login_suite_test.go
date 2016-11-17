package login_test

import (
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
	page     *agouti.Page
	driver   *agouti.WebDriver
	baseURL  = "https://portal.gravitational.io/web/"
	userName = "alexey@gravitational.com"
	password = ""
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
