package e2e

import (
	"github.com/gravitational/robotest/e2e/framework"
	. "github.com/onsi/ginkgo"
)

var _ = framework.RoboDescribe("Application backup and restore [backup][restore]", func() {
	It("should be able to make backup", func() {
		framework.BackupApplication()
	})
	It("should be able to restore application", func() {
		framework.RestoreApplication()
	})
})
