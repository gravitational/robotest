package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"
)

// TODO
type failoverParam struct {
	installParam
}

// failover tests master failover
func failover(p interface{}) (gravity.TestFunc, error) {
	param := p.(failoverParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		cluster, err := provisionNodes(g, cfg, param.installParam)
		g.OK("VMs ready", err)
		defer func() {
			g.Maybe("destroy", cluster.Destroy())
		}()

		installerURL := cfg.InstallerURL
		if param.InstallerURL != "" {
			installerURL = param.InstallerURL
		}

		g.OK("installer downloaded", g.SetInstaller(cluster.Nodes, installerURL, "install"))
		if param.Script != nil {
			g.OK("post bootstrap script",
				g.ExecScript(cluster.Nodes, param.Script.Url, param.Script.Args))
		}
		g.OK("application installed", g.OfflineInstall(cluster.Nodes, param.InstallParam))
		g.OK("status", g.Status(cluster.Nodes))
		g.OK("master failover", g.Failover(cluster.Nodes))
	}, nil
}
