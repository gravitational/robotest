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

// driver is a test-global web driver instance
var driver *web.WebDriver

// New creates a new instance of the framework.
// Creating a framework instance installs a set of BeforeEach/AfterEach to
// emulate BeforeAll/AfterAll for controlled access to resources that should
// only be created once per context
func New() *T {
	f := &T{}

	BeforeEach(f.BeforeEach)
	AfterEach(f.AfterEach)

	return f
}

// T defines a framework type.
// Framework stores attributes common to a single context
type T struct {
	Page *web.Page
}

// BeforeEach emulates BeforeAll for a context.
// It creates a new web page that is only initialized once per series of It
// grouped in any given context
func (r *T) BeforeEach() {
	if r.Page == nil {
		var err error
		r.Page, err = newPage()
		Expect(err).NotTo(HaveOccurred())
	}
}

func (r *T) AfterEach() {
}

// CreateDriver creates a new instance of the web driver
func CreateDriver() {
	if TestContext.WebDriverURL != "" {
		log.Debugf("WebDriverURL specified - skip CreateDriver")
		return
	}
	driver = web.ChromeDriver()
	Expect(driver).NotTo(BeNil())
	Expect(driver.Start()).To(Succeed())
}

// CloseDriver stops and closes the test-global web driver
func CloseDriver() {
	if driver != nil {
		Expect(driver.Stop()).To(Succeed())
	}
}

// Distribute executes the specified command on nodes
func Distribute(command string, nodes ...infra.Node) {
	Expect(Cluster).NotTo(BeNil(), "requires a cluster")
	Expect(Cluster.Provisioner()).NotTo(BeNil(), "requires a provisioner")
	if len(nodes) == 0 {
		nodes = Cluster.Provisioner().NodePool().AllocatedNodes()
	}
	Expect(infra.Distribute(command, nodes...)).To(Succeed())
}

// Cluster is the global instance of the cluster the tests are executed on
var Cluster infra.Infra

// installerNode is the node with installer running on it in case the tests
// are running in wizard mode
var installerNode infra.Node

// InstallerNode returns the node with the running installer.
// Only applicable in wizard mode (TestContext.Wizard == true)
func InstallerNode() infra.Node {
	return installerNode
}

// InitializeCluster creates infrastructure according to configuration
func InitializeCluster() {
	config := infra.Config{ClusterName: TestContext.ClusterName}

	var err error
	var provisioner infra.Provisioner
	if testState != nil {
		if testState.Provisioner != "" {
			provisioner, err = provisionerFromState(config, *testState)
			Expect(err).NotTo(HaveOccurred())
		}
	} else if !TestContext.Teardown {
		TestContext.StateDir, err = newStateDir(TestContext.ClusterName)
		Expect(err).NotTo(HaveOccurred())
		if TestContext.Provisioner != "" {
			provisioner, err = provisionerFromConfig(config, TestContext.StateDir, TestContext.Provisioner)
			Expect(err).NotTo(HaveOccurred())

			installerNode, err = provisioner.Create()
			Expect(err).NotTo(HaveOccurred())
		}
	}

	switch {
	case testState == nil:
		log.Debug("init test state")
		// TestContext -> testState
		testState = &TestState{
			EntryURL: TestContext.OpsCenterURL,
			StateDir: TestContext.StateDir,
		}
		if TestContext.Application.Locator != nil {
			testState.Application = TestContext.Application.Locator
		}
		if provisioner != nil {
			testState.Provisioner = TestContext.Provisioner
			provisionerState := provisioner.State()
			testState.ProvisionerState = &provisionerState
		}
		// Save initial state as soon as possible
		Expect(saveState(withoutBackup)).To(Succeed())
	case testState != nil:
		if provisioner != nil && testState.ProvisionerState.InstallerAddr != "" {
			// Get reference to installer node if the cluster was provisioned with installer
			installerNode, err = provisioner.NodePool().Node(testState.ProvisionerState.InstallerAddr)
			Expect(err).NotTo(HaveOccurred())
		}
	}

	var application *loc.Locator
	if mode == wizardMode {
		Cluster, application, err = infra.NewWizard(config, provisioner, installerNode)
		TestContext.Application.Locator = application
	} else {
		Cluster, err = infra.New(config, TestContext.OpsCenterURL, provisioner)
	}
	Expect(err).NotTo(HaveOccurred())
	TestContext.OpsCenterURL = Cluster.OpsCenterURL()

}

// Destroy destroys the infrastructure created previously in InitializeCluster
// and removes state directory
func Destroy() {
	if Cluster != nil {
		Expect(Cluster.Close()).To(Succeed())
		Expect(Cluster.Destroy()).To(Succeed())
	}

	if testState == nil {
		log.Debug("no test state read: skip state clean up")
		return
	}
	// Clean up state
	err := os.Remove(stateConfigFile)
	if err != nil && !os.IsNotExist(err) {
		Failf("failed to remove state file %q: %v", stateConfigFile, err)
	}
	err = system.RemoveAll(TestContext.StateDir)
	if err != nil && !os.IsNotExist(err) {
		Failf("failed to cleanup state directory %q: %v", TestContext.StateDir, err)
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
	if Cluster.Provisioner() != nil {
		provisionerState := Cluster.Provisioner().State()
		testState.ProvisionerState = &provisionerState
	}

	Expect(saveState(withBackup)).To(Succeed())
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

	opsURL := TestContext.OpsCenterURL
	err := ConnectToOpsCenter(opsURL, TestContext.ServiceLogin)
	if err != nil {
		// If connect to Ops Center fails, no site report can be collected
		// so bail out
		log.Errorf("failed to connect to the Ops Center %q: %v", opsURL, err)
		return
	}

	tempOutput, err := ioutil.TempFile("", "crashreport")
	Expect(err).NotTo(HaveOccurred(), "expected to create a temporary file")
	defer func() {
		tempOutput.Close()
		if err := os.Remove(tempOutput.Name()); err != nil {
			log.Warningf("failed to remove temporary report file %q: %v", tempOutput.Name(), err)
		}
	}()

	output := filepath.Join(TestContext.ReportDir, "crashreport.tar.gz")
	stateDir := fmt.Sprintf("--state-dir=%v", TestContext.StateDir)
	opsURL = fmt.Sprintf("--ops-url=%v", Cluster.OpsCenterURL())
	cmd := exec.Command("gravity", "--insecure", stateDir, "site", "report", opsURL, TestContext.ClusterName, tempOutput.Name())
	err = system.Exec(cmd, io.MultiWriter(os.Stderr, GinkgoWriter))
	if err != nil {
		log.Errorf("failed to collect site report: %v", err)
	} else {
		Expect(system.CopyFile(output, tempOutput.Name())).To(Succeed())
	}

	if Cluster.Provisioner() == nil {
		log.Infof("no provisioner: skip collecting provisioner logs")
		return
	}

	if installerNode != nil {
		// Collect installer log
		installerLog, err := os.Create(filepath.Join(TestContext.ReportDir, "installer.log"))
		Expect(err).NotTo(HaveOccurred())
		defer installerLog.Close()

		err = infra.ScpText(installerNode, Cluster.Provisioner().InstallerLogPath(), installerLog)
		if err != nil {
			log.Errorf("failed to fetch the installer log from %q: %v", installerNode, err)
			os.Remove(installerLog.Name())
		}
	}
	for _, node := range Cluster.Provisioner().NodePool().Nodes() {
		agentLog, err := os.Create(filepath.Join(TestContext.ReportDir,
			fmt.Sprintf("agent_%v.log", node.Addr())))
		Expect(err).NotTo(HaveOccurred())
		defer agentLog.Close()
		errCopy := infra.ScpText(node, defaults.AgentLogPath, agentLog)
		if errCopy != nil {
			log.Errorf("failed to fetch agent log from %s: %v", node, errCopy)
		}
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

func saveState(withBackup backupFlag) error {
	if withBackup {
		filename := fmt.Sprintf("%vbackup", filepath.Base(stateConfigFile))
		stateConfigBackup := filepath.Join(filepath.Dir(stateConfigFile), filename)
		err := system.CopyFile(stateConfigBackup, stateConfigFile)
		if err != nil {
			log.Errorf("failed to make a backup of state file %q: %v", stateConfigFile, err)
		}
	}

	file, err := os.Create(stateConfigFile)
	if err != nil {
		return trace.Wrap(err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	err = enc.Encode(testState)
	if err != nil {
		return trace.Wrap(err)
	}

	return nil
}

func newPage() (*web.Page, error) {
	if TestContext.WebDriverURL != "" {
		capabilities := web.NewCapabilities().Browser("chrome").Platform("linux").With("javascriptEnabled")
		return web.NewPage(TestContext.WebDriverURL, web.Desired(capabilities))
	}
	return driver.NewPage()
}

func newStateDir(clusterName string) (dir string, err error) {
	dir, err = ioutil.TempDir("", fmt.Sprintf("robotest-%v-", clusterName))
	if err != nil {
		return "", trace.Wrap(err)
	}
	log.Infof("state directory: %v", dir)
	return dir, nil
}

type backupFlag bool

const (
	withBackup    backupFlag = true
	withoutBackup backupFlag = false
)
