package e2e

import (
	"os"
	"testing"

	"github.com/gravitational/robotest/e2e/framework"
	uidefaults "github.com/gravitational/robotest/e2e/uimodel/defaults"
	"github.com/gravitational/robotest/lib/constants"

	log "github.com/Sirupsen/logrus"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

// TestE2E runs e2e tests using the ginkgo runner.
// If a TestContext.ReportDir is specified, cluster logs will also be saved.
func RunE2ETests(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	if framework.TestContext.ReportDir != "" {
		errCreate := os.MkdirAll(framework.TestContext.ReportDir, constants.SharedDirMask)
		if errCreate != nil {
			framework.Failf("failed to create report directory %q: %v", errCreate)
		}
		log.Infof("report directory: %q", framework.TestContext.ReportDir)
	}
	gomega.SetDefaultEventuallyPollingInterval(uidefaults.EventuallyPollInterval)
	ginkgo.RunSpecs(t, "Robotest e2e suite")
}

// Run the tasks that are meant to be run once per invocation
var _ = ginkgo.SynchronizedBeforeSuite(func() []byte {
	// Run only on ginkgo node 1
	framework.CreateDriver()
	framework.InitializeCluster()
	return nil
}, func([]byte) {
})

var _ = ginkgo.SynchronizedAfterSuite(func() {
	// Run on all ginkgo nodes
}, func() {
	// Run only on ginkgo node 1
	if framework.TestContext.DumpCore {
		framework.CoreDump()
		return
	}

	if !framework.TestContext.Teardown {
		framework.UpdateState()
	}
	framework.CloseDriver()
	if framework.TestContext.Teardown {
		if framework.TestContext.ReportDir != "" {
			framework.CoreDump()
		}
		framework.Destroy()
	}
})
