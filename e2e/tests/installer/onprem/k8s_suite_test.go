package onprem

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gravitational/robotest/e2e/ui"
	"github.com/sclevine/agouti"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/vagrant"
)

func TestK8s(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "onprem")
}

var (
	driver      *agouti.WebDriver
	page        *agouti.Page
	provisioner infra.Provisioner
)

var (
	userName   = "alex@kontsevoy.com"
	password   = "test123"
	domainName = "alexeyk-11-11"
	startURL   = "https://localhost:8080/web/installer/new/gravitational.io/k8s-onprem/0.45.14-138"
)

var _ = BeforeSuite(func() {
	InitProvisioner()
	InitDriver()
	// opens a page and if login is required attemps to sign-in
	ui.EnsureUser(page, startURL, userName, password, ui.WithEmail)
})

var _ = AfterSuite(func() {
	Expect(driver.Stop()).To(Succeed())

	if provisioner != nil {
		Expect(provisioner.Destroy()).To(Succeed())
	}
})

func InitDriver() {
	var err error
	driver = agouti.ChromeDriver()
	Expect(driver.Start()).To(Succeed())

	page, err = driver.NewPage()
	Expect(err).NotTo(HaveOccurred())
}

func InitProvisioner() {
	var err error

	config := infra.Config{
		OpsCenterURL: "",
		ClusterName:  "test",
	}

	provisioner, err = vagrant.New("/home/akontsevoy/onprem/test", vagrant.Config{
		Nodes:        1,
		Config:       config,
		ScriptPath:   "/home/akontsevoy/go/src/github.com/gravitational/robotest/e2e/tests/tmp/Vagrantfile",
		InstallerURL: "path/to/installer.tar.gz",
	})

	Expect(err).ShouldNot(HaveOccurred())

	_, err = infra.New(config, provisioner)

	Expect(err).ShouldNot(HaveOccurred())
}
