/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"os"
	"testing"

	"github.com/gravitational/robotest/e2e/framework"
	uidefaults "github.com/gravitational/robotest/e2e/uimodel/defaults"
	"github.com/gravitational/robotest/lib/constants"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

// TestE2E runs e2e tests using the ginkgo runner.
// If a TestContext.ReportDir is specified, cluster logs will also be saved.
func RunE2ETests(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	if framework.TestContext.ReportDir != "" {
		errCreate := os.MkdirAll(framework.TestContext.ReportDir, constants.SharedDirMask)
		if errCreate != nil {
			framework.Failf("Failed to create report directory %q: %v",
				framework.TestContext.ReportDir, errCreate)
		}
		log.WithField("dir", framework.TestContext.ReportDir).Info("New report directory.")
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
