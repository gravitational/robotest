package specs

import (
	"github.com/gravitational/robotest/e2e/framework"
	. "github.com/onsi/ginkgo"
)

func VerifyBackup(f *framework.T) {
	var _ = framework.RoboDescribe("Application Backup", func() {

		It("should be able to make backup", func() {
			framework.BackupApplication()
		})
	})
}

func VerifyRestore(f *framework.T) {
	var _ = framework.RoboDescribe("Application Restore", func() {

		It("should be able to restore application", func() {
			framework.RestoreApplication()
		})
	})
}
