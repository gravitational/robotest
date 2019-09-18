package sanity

import (
	"fmt"

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

		leader, err := g.GetLeaderNode(cluster.Nodes)
		g.OK(fmt.Sprintf("leader=%v", leader), err)
		g.OK("disconnect leader", g.Disconnect(leader))

		leader, err = g.GetLeaderNode(cluster.Nodes)
		g.OK(fmt.Sprintf("new leader=%v", leader), err)

		g.OK("reconnect previous leader", g.Connect(leader))
		g.OK("status", g.Status(cluster.Nodes))
	}, nil
}
