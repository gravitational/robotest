package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"
)

type upgradeParam struct {
	installParam
	// BaseInstallerURL is initial app installer URL
	BaseInstallerURL string `json:"from" validate:"required"`
}

func upgrade(p interface{}) (gravity.TestFunc, error) {
	param := p.(upgradeParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		nodes, destroyFn, err := provisionNodes(g, cfg, param.installParam)
		g.OK("provision nodes", err)
		defer destroyFn()

		g.OK("base installer", g.SetInstaller(nodes, param.BaseInstallerURL, "base"))
		g.OK("install", g.OfflineInstall(nodes, param.InstallParam))
		g.OK("status", g.Status(nodes))
		g.OK("upgrade", g.Upgrade(nodes, cfg.InstallerURL, "upgrade"))
		g.OK("status", g.Status(nodes))
	}, nil
}
