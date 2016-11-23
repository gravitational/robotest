package framework

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/lib/defaults"
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

func InitializeCluster() {
	config := infra.Config{ClusterName: TestContext.ClusterName}

	var err error
	var stateDir string
	var provisioner infra.Provisioner
	if testState != nil {
		provisioner, err = provisionerFromState(config, *testState)
		Expect(err).NotTo(HaveOccurred())
	} else {
		if TestContext.Provisioner != "" {
			stateDir, err = newStateDir(TestContext.ClusterName)
			Expect(err).NotTo(HaveOccurred())

			provisioner, err = provisionerFromConfig(config, stateDir, TestContext.Provisioner)
			Expect(err).NotTo(HaveOccurred())

			installerNode, err = provisioner.Create()
			Expect(err).NotTo(HaveOccurred())
		}
	}

	var application *loc.Locator
	if TestContext.Wizard {
		Cluster, application, err = infra.NewWizard(config, provisioner, installerNode)
		TestContext.Application = application
	} else {
		Cluster, err = infra.New(config, TestContext.OpsCenterURL, provisioner)
	}
	Expect(err).NotTo(HaveOccurred())

	if testState != nil && TestContext.Onprem.InstallerURL != "" {
		// Get reference to installer node if the cluster was provisioned with installer
		installerNode, err = Cluster.Provisioner().Node(testState.ProvisionerState.InstallerAddr)
		Expect(err).NotTo(HaveOccurred())
	}

	if Cluster.Provisioner() != nil && testState == nil {
		log.Debug("init test state")
		testState = &TestState{
			OpsCenterURL:     Cluster.OpsCenterURL(),
			Provisioner:      TestContext.Provisioner,
			ProvisionerState: Cluster.Provisioner().State(),
			StateDir:         stateDir,
		}
		Expect(saveState()).To(Succeed())
	}
}

func Destroy() {
	if Cluster != nil {
		Expect(Cluster.Close()).To(Succeed())
		Expect(Cluster.Destroy()).To(Succeed())
	}
	// Clean up state
	err := os.Remove(stateConfigFile)
	if err != nil && !os.IsNotExist(err) {
		Failf("failed to remove state file %q: %v", stateConfigFile, err)
	}
	if testState == nil {
		return
	}
	err = system.RemoveContents(testState.StateDir)
	if err != nil && !os.IsNotExist(err) {
		Failf("failed to remove state directory %q: %v", testState.StateDir, err)
	}
}

// UpdateState updates the state file with the current provisioner state.
// It validates the context to avoid updating a state file on an inactive
// or automatically provisioned cluster
func UpdateState() {
	if Cluster == nil || testState == nil {
		log.Infof("cluster inactive: skip UpdateState")
		return
	}
	if Cluster.Provisioner() == nil {
		log.Infof("cluster is auto-provisioned: skip UpdateState")
		return
	}

	testState.ProvisionerState = Cluster.Provisioner().State()
	Expect(saveState()).To(Succeed())
}

// CoreDump collects diagnostic information into the specified report directory
// after the tests
func CoreDump() {
	if Cluster == nil {
		log.Infof("cluster inactive: skip CoreDump")
		return
	}
	if TestContext.ServiceLogin.IsEmpty() {
		log.Infof("no service login configured: skip CoreDump")
		return
	}
	// TODO: use a temporary state directory to avoid clashes with
	// existing services
	cmd := exec.Command("gravity", "ops", "connect", Cluster.OpsCenterURL(),
		TestContext.ServiceLogin.Username, TestContext.ServiceLogin.Password)
	err := system.Exec(cmd, io.MultiWriter(os.Stderr, GinkgoWriter))
	if err != nil {
		// If connect to Ops Center fails, no site report can be collected
		// so bail out
		log.Errorf("failed to connect to the Ops Center: %v", err)
		return
	}

	output := filepath.Join(TestContext.ReportDir, "crashreport.tar.gz")
	opsURL := fmt.Sprintf("--ops-url=%v", Cluster.OpsCenterURL())
	cmd = exec.Command("gravity", "--insecure", "site", "report", opsURL, TestContext.ClusterName, output)
	err = system.Exec(cmd, io.MultiWriter(os.Stderr, GinkgoWriter))
	if err != nil {
		log.Errorf("failed to collect site report: %v", err)
	}

	if Cluster.Provisioner() == nil {
		return
	}

	if installerNode != nil {
		// Collect installer log
		installerLog, err := os.Create(filepath.Join(TestContext.ReportDir, "installer.log"))
		Expect(err).NotTo(HaveOccurred())
		defer installerLog.Close()

		Expect(infra.ScpText(installerNode,
			Cluster.Provisioner().InstallerLogPath(), installerLog)).To(Succeed())
	}
	for _, node := range Cluster.Provisioner().AllNodes() {
		agentLog, err := os.Create(filepath.Join(TestContext.ReportDir,
			fmt.Sprintf("agent_%v.log", node.Addr())))
		Expect(err).NotTo(HaveOccurred())
		defer agentLog.Close()
		Expect(infra.ScpText(node, defaults.AgentLogPath, agentLog)).To(Succeed())
		// TODO: collect shrink operation agent logs
	}
}

// RoboDescribe is local wrapper function for ginkgo.Describe.
// It adds test namespacing.
// TODO: eventually benefit from safe test tags: https://github.com/kubernetes/kubernetes/pull/22401.
func RoboDescribe(text string, body func()) bool {
	return Describe("[robotest] "+text, body)
}

// RunAgentCommand interprets the specified command as agent command.
// It will modify the agent command line to start agent in background
// and will distribute the command on the specified nodes
func RunAgentCommand(command string, nodes ...infra.Node) {
	command, err := infra.ConfigureAgentCommandRunDetached(command)
	Expect(err).NotTo(HaveOccurred())
	Distribute(command, nodes...)
}

func saveState() error {
	file, err := os.Create(stateConfigFile)
	if err != nil {
		return trace.Wrap(err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return trace.Wrap(enc.Encode(testState))
}

func newStateDir(clusterName string) (dir string, err error) {
	dir, err = ioutil.TempDir("", fmt.Sprintf("robotest-%v-", clusterName))
	if err != nil {
		return "", trace.Wrap(err)
	}
	log.Infof("state directory: %v", dir)
	return dir, nil
}
