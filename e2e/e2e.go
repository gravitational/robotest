package e2e

import (
	"os"
	"testing"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/lib/defaults"

	log "github.com/Sirupsen/logrus"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

// TestE2E runs E tests using the ginkgo runner.
// If a TestContext.ReportDir is specified, cluster logs will also be saved.
func RunE2ETests(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	if framework.TestContext.ReportDir != "" {
		errCreate := os.MkdirAll(framework.TestContext.ReportDir, defaults.SharedDirMask)
		if errCreate != nil {
			framework.Failf("failed to create report directory %q: %v", errCreate)
		}
		log.Infof("report directory: %q", framework.TestContext.ReportDir)
	}
	ginkgo.RunSpecs(t, "Robotest e2e suite")
}

// Run the tasks that are meant to be run once per invocation
var _ = ginkgo.SynchronizedBeforeSuite(func() []byte {
	// Run only ginkgo on node 1
	// TODO
	// setupCluster()
	log.Infof("In BeforeSuite")
	return nil
}, func([]byte) {
})

var _ = ginkgo.SynchronizedAfterSuite(func() {
	// Run on all ginkgo nodes
}, func() {
	log.Infof("In AfterSuite")
	// Run only ginkgo on node 1
	if framework.TestContext.ReportDir != "" {
		framework.CoreDump()
	}
})
