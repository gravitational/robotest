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
