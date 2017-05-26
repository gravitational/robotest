package e2e

import (
	"github.com/gravitational/robotest/e2e/framework"
	. "github.com/onsi/ginkgo"
)

var _ = framework.RoboDescribe("Application backup and restore", func() {
	It("should be able to backup [backup]", func() {
		framework.BackupApplication()
	})
	It("should be able to restore [restore]", func() {
		framework.RestoreApplication()
	})
})
