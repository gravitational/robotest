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

func Distribute(command string, nodes ...infra.Node) {
	Expect(Cluster).NotTo(BeNil(), "requires a cluster")
	Expect(Cluster.Provisioner()).NotTo(BeNil(), "requires a provisioner")
	if len(nodes) == 0 {
		nodes = Cluster.Provisioner().Nodes()
	}
	Expect(infra.Distribute(command, nodes...)).To(Succeed())
}

// Cluster is the global instance of the cluster the tests are executed on
var Cluster infra.Infra

// installerNode is the node with installer running on it in case the tests
// are running in wizard mode
var installerNode infra.Node

func SetupCluster() {
	config := infra.Config{ClusterName: TestContext.ClusterName}

	var provisioner infra.Provisioner
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
	cmd := exec.Command("gravity", "ops", "connect", Cluster.OpsCenterURL(),
		TestContext.Login.Username, TestContext.Login.Password)
	Expect(system.Exec(cmd, io.MultiWriter(os.Stderr, GinkgoWriter))).To(Succeed())

	output := filepath.Join(TestContext.ReportDir, "crashreport.tar.gz")
	opsURL := fmt.Sprintf("--ops-url=%v", Cluster.OpsCenterURL())
	cmd = exec.Command("gravity", "--insecure", "site", "report", opsURL, TestContext.ClusterName, output)
	Expect(system.Exec(cmd, io.MultiWriter(os.Stderr, GinkgoWriter))).To(Succeed())

	// TODO: this implies a test run (incl. infra setup) per invocation
	// Since this is headed in the direction of shared state, installer node should also
	// be persisted as state attribute
	if installerNode != nil {
		// Collect installer log
		installerLog, err := os.Create(filepath.Join(TestContext.ReportDir, "installer.log"))
		Expect(err).NotTo(HaveOccurred())
		defer installerLog.Close()

		Expect(infra.ScpText(installerNode,
			Cluster.Provisioner().InstallerLogPath(), installerLog)).To(Succeed())
	}

	for _, node := range Cluster.Provisioner().Nodes() {
		agentLog, err := os.Create(filepath.Join(TestContext.ReportDir,
			fmt.Sprintf("agent_%v.log", node.Addr())))
		Expect(err).NotTo(HaveOccurred())
		defer agentLog.Close()
		Expect(infra.ScpText(node,
			"/var/log/gravity.agent.log", agentLog)).To(Succeed())
		// TODO: collect other operations agent logs from nodes
	}
}

// RoboDescribe is local wrapper function for ginkgo.Describe.
// It adds test namespacing.
// TODO: eventually benefit from safe test tags: https://github.com/kubernetes/kubernetes/pull/22401.
func RoboDescribe(text string, body func()) bool {
	return Describe("[robotest] "+text, body)
}

func RunAgentCommand(command string, nodes ...infra.Node) {
	command, err := infra.ConfigureAgentCommandRunDetached(command)
	Expect(err).NotTo(HaveOccurred())
	Distribute(command, nodes...)
}

func newStateDir(clusterName string) (dir string, err error) {
	dir, err = ioutil.TempDir("", fmt.Sprintf("robotest-%v-", clusterName))
	if err != nil {
		return "", trace.Wrap(err)
	}
	log.Infof("state directory: %v", dir)
	return dir, nil
}
