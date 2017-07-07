package sanity

import (
	"fmt"

	"github.com/gravitational/robotest/infra/gravity"
)

type resizeParam struct {
	installParam
	// TargetNodes is how many nodes cluster should have after expand
	ToNodes uint `json:"to" validate:"required,gte=3"`
}

// resize installs an initial cluster and then expands or gracefully shrinks it to given number of nodes
func resize(p interface{}) (gravity.TestFunc, error) {
	param := p.(resizeParam)

	return func(g *gravity.TestContext, baseConfig gravity.ProvisionerConfig) {
		config := baseConfig.WithNodes(param.ToNodes)

		nodes, destroyFn, err := g.Provision(config)
		g.OK("provision nodes", err)
		defer destroyFn()

		g.OK("download installer", g.SetInstaller(nodes, config.InstallerURL, "install"))
		g.OK(fmt.Sprintf("install on %d node", param.NodeCount),
			g.OfflineInstall(nodes[0:param.NodeCount], param.InstallParam))
		g.OK("status", g.Status(nodes[0:param.NodeCount]))
		g.OK("time sync", g.CheckTimeSync(nodes))

		g.OK(fmt.Sprintf("expand to %d nodes", param.ToNodes),
			g.Expand(nodes[0:param.NodeCount], nodes[param.NodeCount:param.ToNodes], param.Role))
		g.OK("status", g.Status(nodes[0:param.ToNodes]))
	}, nil
}
