package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"
)

type installParam struct {
	gravity.InstallParam
	// NodeCount is how many nodes
	NodeCount uint `json:"nodes" validate:"gte=1"`
	// Timeout defines operation timeouts
	Timeouts gravity.OpTimeouts
}

func install(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(g gravity.TestContext, baseConfig gravity.ProvisionerConfig) {
		cfg := baseConfig.WithNodes(param.NodeCount)
		nodes, destroyFn, err := g.Provision(cfg)
		g.OK("provision nodes", err)
		defer destroyFn()

		g.OK("download installer", g.SetInstaller(nodes, cfg.InstallerURL, "install"))
		g.OK("install", g.OfflineInstall(nodes, param.InstallParam))
		g.OK("status", g.Status(nodes))
	}, nil
}

func provision(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(g gravity.TestContext, baseConfig gravity.ProvisionerConfig) {
		cfg := baseConfig.WithNodes(param.NodeCount)
		nodes, destroyFn, err := g.Provision(cfg)
		g.OK("provision nodes", err)
		defer destroyFn()

		g.OK("download installer", g.SetInstaller(nodes, cfg.InstallerURL, "install"))
	}, nil
}
