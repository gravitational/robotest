package e2e

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/specs"
)

var _ = framework.RoboDescribe("Site Update Test", func() {
	f := framework.New()
	specs.VerifySiteUpdate(f)
})
