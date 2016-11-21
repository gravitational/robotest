package framework

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/lib/loc"
	"github.com/gravitational/robotest/lib/system"
	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
)

var driver *web.WebDriver

func New() *T {
	f := &T{}

	BeforeEach(f.BeforeEach)
	AfterEach(f.AfterEach)

	return f
}

type T struct {
	Page *web.Page
}

func (r *T) BeforeEach() {
	if r.Page == nil {
		var err error
		r.Page, err = driver.NewPage()
		Expect(err).NotTo(HaveOccurred())
	}
}

func (r *T) AfterEach() {
}

func CreateDriver() {
	driver = web.ChromeDriver()
	Expect(driver).NotTo(BeNil())
	Expect(driver.Start()).To(Succeed())
}

func CloseDriver() {
	Expect(driver.Stop()).To(Succeed())
}

func Distribute(command string) {
	Expect(Cluster).NotTo(BeNil(), "requires a cluster")
	Expect(Cluster.Provisioner()).NotTo(BeNil(), "requires a provisioner")
	Expect(infra.Distribute(command, Cluster.Provisioner().Nodes())).To(Succeed())
}

// Cluster is the global instance of the cluster the tests are executed on
var Cluster infra.Infra

func SetupCluster() {
	config := infra.Config{ClusterName: TestContext.ClusterName}

	var provisioner infra.Provisioner
	var installerNode infra.Node
	if TestContext.Provisioner != "" {
		stateDir, err := newStateDir(TestContext.ClusterName)
		Expect(err).NotTo(HaveOccurred())

		provisioner, err = provisionerFromConfig(config, stateDir, TestContext.Provisioner)
		Expect(err).NotTo(HaveOccurred())

		installerNode, err = provisioner.Create()
		Expect(err).NotTo(HaveOccurred())
	}

	var err error
	var application *loc.Locator
	if TestContext.Wizard {
		Cluster, application, err = infra.NewWizard(config, provisioner, installerNode)
		TestContext.Application = application
	} else {
		Cluster, err = infra.New(config, TestContext.OpsCenterURL, provisioner)
	}
	Expect(err).NotTo(HaveOccurred())
}

func DestroyCluster() {
	if Cluster != nil {
		Expect(Cluster.Close()).To(Succeed())
		Expect(Cluster.Destroy()).To(Succeed())
	}
}

// CoreDump collects diagnostic information into the specified report directory
// after the tests
func CoreDump() {
	if Cluster == nil {
		log.Infof("cluster inactive: skip CoreDump")
		return
	}
	output := filepath.Join(TestContext.ReportDir, "crashreport.tar.gz")
	opsURL := fmt.Sprintf("--ops-url=%v", Cluster.OpsCenterURL())
	cmd := exec.Command("gravity", "ops", "connect", Cluster.OpsCenterURL(),
		TestContext.Login.Username, TestContext.Login.Password)
	Expect(system.Exec(cmd, io.MultiWriter(os.Stderr, GinkgoWriter))).To(Succeed())

	cmd = exec.Command("gravity", "--insecure", "site", "report", opsURL, TestContext.ClusterName, output)
	Expect(system.Exec(cmd, io.MultiWriter(os.Stderr, GinkgoWriter))).To(Succeed())
}

// RoboDescribe is local wrapper function for ginkgo.Describe.
// It adds test namespacing.
// TODO: eventually benefit from safe test tags: https://github.com/kubernetes/kubernetes/pull/22401.
func RoboDescribe(text string, body func()) bool {
	return Describe("[robotest] "+text, body)
}

func RunAgentCommand(command string) {
	command, err := infra.ConfigureAgentCommandRunDetached(command)
	Expect(err).NotTo(HaveOccurred())
	Distribute(command)
}

func newStateDir(clusterName string) (dir string, err error) {
	dir, err = ioutil.TempDir("", fmt.Sprintf("robotest-%v-", clusterName))
	if err != nil {
		return "", trace.Wrap(err)
	}
	log.Infof("state directory: %v", dir)
	return dir, nil
}
