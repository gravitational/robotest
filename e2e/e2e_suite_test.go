package e2e

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/ui"
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/vagrant"
	"github.com/gravitational/robotest/lib/system"
	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
)

func init() {
	initLogger(log.DebugLevel)
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "e2e")
}

var (
	driver   *agouti.WebDriver
	page     *agouti.Page
	cluster  infra.Infra
	stateDir string
)

var _ = BeforeSuite(func() {
	var err error
	stateDir, err = newStateDir(framework.TestContext.ClusterName)
	Expect(err).NotTo(HaveOccurred())

	//initCluster()
	initDriver()
	// Navigate to the starting URL and login if necessary
	ui.EnsureUser(page, framework.TestContext.StartURL,
		framework.TestContext.Login.Username,
		framework.TestContext.Login.Password, ui.WithGoogle)
})

var _ = AfterSuite(func() {
	if driver != nil {
		Expect(driver.Stop()).To(Succeed())
	}

	if cluster != nil {
		Expect(cluster.Close()).To(Succeed())
		Expect(cluster.Destroy()).To(Succeed())
		Expect(system.RemoveContents(stateDir)).To(Succeed())
	}
})

func getPage() *agouti.Page {
	return page
}

func initDriver() {
	var err error
	driver = agouti.ChromeDriver()
	Expect(driver.Start()).To(Succeed())

	page, err = driver.NewPage()
	Expect(err).NotTo(HaveOccurred())
}

func initCluster() {
	var err error

	var provisioner infra.Provisioner
	config := infra.Config{ClusterName: framework.TestContext.ClusterName}
	provisioner, err = vagrant.New(stateDir, vagrant.Config{
		Config:       config,
		ScriptPath:   framework.TestContext.Onprem.ScriptPath,
		InstallerURL: framework.TestContext.Onprem.InstallerURL,
	})
	Expect(err).ShouldNot(HaveOccurred())

	cluster, err = infra.New(config, framework.TestContext.OpsCenterURL, provisioner)
	Expect(err).ShouldNot(HaveOccurred())
}

func newStateDir(clusterName string) (dir string, err error) {
	dir, err = ioutil.TempDir("", fmt.Sprintf("robotest-%v-", clusterName))
	if err != nil {
		return "", trace.Wrap(err)
	}
	log.Infof("state directory: %v", dir)
	return dir, nil
}

func initLogger(level log.Level) {
	log.StandardLogger().Hooks = make(log.LevelHooks)
	log.SetFormatter(&trace.TextFormatter{TextFormatter: log.TextFormatter{FullTimestamp: true}})
	log.SetOutput(os.Stderr)
	log.SetLevel(level)
}
