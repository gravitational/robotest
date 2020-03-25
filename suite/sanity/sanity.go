package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/robotest/lib/config"
	"github.com/gravitational/robotest/lib/defaults"
)

var defaultInstallParam = installParam{
	InstallParam: gravity.InstallParam{
		StateDir: defaults.GravityDir,
	},
}

// Suite returns base configuration for a suite which may be further customized
func Suite() *config.Config {
	cfg := config.New()

	cfg.Add("noop", noop, noopParam{})
	cfg.Add("noopV", noopVariety, noopParam{})
	cfg.Add("provision", provision, defaultInstallParam)
	cfg.Add("resize", resize, resizeParam{installParam: defaultInstallParam})
	cfg.Add("install", install, defaultInstallParam)
	cfg.Add("recover", lossAndRecovery, lossAndRecoveryParam{installParam: defaultInstallParam})
	cfg.Add("recoverV", lossAndRecoveryVariety, defaultInstallParam)
	cfg.Add("upgrade", upgrade, upgradeParam{installParam: defaultInstallParam})
	cfg.Add("autoscale", autoscale, defaultInstallParam)

	return cfg
}
