package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/robotest/lib/config"
	"github.com/gravitational/robotest/lib/defaults"
)

var defaultProvisionParam = provisionParam{
	InstallParam: gravity.InstallParam{
		StateDir: defaults.GravityDir,
	},
}

var defaultInstallParam = installParam{
	provisionParam: defaultProvisionParam,
}

const (
	checkpointProvision = "provision"
	checkpointInstall   = "install"
)

// Suite returns base configuration for a suite which may be further customized
func Suite() *config.Config {
	cfg := config.New()

	cfg.Add("noop", noop, noopParam{})
	cfg.Add("noopV", noopVariety, noopParam{})
	cfg.Add("provision", provision, defaultProvisionParam)
	cfg.Add("resize", resize, resizeParam{installParam: defaultInstallParam})
	cfg.Add("install", install, defaultInstallParam)
	cfg.Add("recover", lossAndRecovery, lossAndRecoveryParam{installParam: defaultInstallParam})
	cfg.Add("recoverV", lossAndRecoveryVariety, defaultInstallParam)
	cfg.Add("upgrade3lts", upgrade, upgradeParam{installParam: defaultInstallParam})
	cfg.Add("autoscale", autoscale, defaultInstallParam)

	return cfg
}
