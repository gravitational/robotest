package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"
)

type installParam struct {
	gravity.InstallParam
	// NodeCount is how many nodes
	NodeCount uint `json:"nodes" validate:"gte=1"`
}

func provisionNodes(g *gravity.TestContext, cfg gravity.ProvisionerConfig, param installParam) ([]gravity.Gravity, gravity.DestroyFn, error) {
	return g.Provision(cfg.WithOS(param.OSFlavor).
		WithStorageDriver(param.DockerStorageDriver).
		WithNodes(param.NodeCount))
}

func install(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		nodes, destroyFn, err := provisionNodes(g, cfg, param)
		g.OK("VMs ready", err)
		defer destroyFn()

		g.OK("installer downloaded", g.SetInstaller(nodes, cfg.InstallerURL, "install"))
		g.OK("application installed", g.OfflineInstall(nodes, param.InstallParam))
		g.OK("status", g.Status(nodes))
	}, nil
}

func provision(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		nodes, destroyFn, err := provisionNodes(g, cfg, param)
		g.OK("provision nodes", err)
		defer destroyFn()

		g.OK("download installer", g.SetInstaller(nodes, cfg.InstallerURL, "install"))
	}, nil
}
