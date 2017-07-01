package sanity

import (
	"github.com/gravitational/robotest/lib/config"

	"github.com/gravitational/robotest/infra/gravity"
)

var defaultTimeouts = gravity.DefaultTimeouts
var defaultInstallParam = installParam{Timeouts: defaultTimeouts, InstallParam: gravity.InstallParam{Role: "node"}}

// Suite returns base configuration for a suite which may be further customized

func Suite() *config.Config {
	cfg := config.New()

	cfg.Add("provision", provision, defaultInstallParam)
	cfg.Add("resize", resize, resizeParam{installParam: defaultInstallParam})
	cfg.Add("install", install, defaultInstallParam)
	cfg.Add("recover", lossAndRecovery, lossAndRecoveryParam{installParam: defaultInstallParam})
	cfg.Add("recover_variety", lossAndRecoveryVariety, defaultInstallParam)
	cfg.Add("upgrade3lts", upgrade, upgradeParam{installParam: defaultInstallParam})

	return cfg
}
