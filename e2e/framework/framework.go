package framework

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/terraform"
	"github.com/gravitational/robotest/infra/vagrant"
	"github.com/gravitational/robotest/lib/loc"
	"github.com/gravitational/robotest/lib/system"
	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// Cluster is the global instance of the cluster the tests are executed on
var Cluster infra.Infra

func SetupCluster() {
	stateDir, err := newStateDir(TestContext.ClusterName)
	Expect(err).NotTo(HaveOccurred())

	config := infra.Config{ClusterName: TestContext.ClusterName}
	provisioner, err := provisionerFromConfig(config, stateDir)
	Expect(err).NotTo(HaveOccurred())

	installerNode, err := provisioner.Create()
	Expect(err).NotTo(HaveOccurred())

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
	output := filepath.Join(TestContext.ReportDir, "crashreport.tar.gz")
	opsURL := fmt.Sprintf("--ops-url=%v", Cluster.OpsCenterURL())
	cmd := exec.Command("gravity", "site", "report", opsURL, TestContext.ClusterName, output)
	err := system.Exec(cmd, io.MultiWriter(os.Stderr, GinkgoWriter))
	if err != nil {
		Failf("failed to collect diagnostics: %v", err)
	}
}

// RoboDescribe is local wrapper function for ginkgo.Describe.
// It adds test namespacing.
// TODO: eventually benefit from safe test tags: https://github.com/kubernetes/kubernetes/pull/22401.
func RoboDescribe(text string, body func()) bool {
	return Describe("[robotest] "+text, body)
}

func provisionerFromConfig(infraConfig infra.Config, stateDir string) (provisioner infra.Provisioner, err error) {
	switch TestContext.Onprem.Provisioner {
	case "terraform":
		config := terraform.Config{
			Config: infraConfig,
		}
		provisioner, err = terraform.New(stateDir, config)
	case "vagrant":
		config := vagrant.Config{
			Config: infraConfig,
		}
		provisioner, err = vagrant.New(stateDir, config)
	default:
		return nil, trace.BadParameter("no provisioner enabled in configuration")
	}
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return provisioner, nil
}

func newStateDir(clusterName string) (dir string, err error) {
	dir, err = ioutil.TempDir("", fmt.Sprintf("robotest-%v-", clusterName))
	if err != nil {
		return "", trace.Wrap(err)
	}
	log.Infof("state directory: %v", dir)
	return dir, nil
}
