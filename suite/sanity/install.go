package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"

	"cloud.google.com/go/bigquery"
)

type installParam struct {
	gravity.InstallParam
	// NodeCount is how many nodes
	NodeCount uint `json:"nodes" validate:"gte=1"`
	// Script if not empty would be executed with args provided after installer has been transferred
	Script *scriptParam `json:"script"`
}

type scriptParam struct {
	Url  string   `json:"url" validate:"required"`
	Args []string `json:"args"`
}

func (p installParam) Save() (row map[string]bigquery.Value, insertID string, err error) {
	row = make(map[string]bigquery.Value)
	row["os"] = p.InstallParam.OSFlavor.Vendor
	row["os_version"] = p.InstallParam.OSFlavor.Version
	row["nodes"] = int(p.NodeCount)
	row["storage"] = p.InstallParam.DockerStorageDriver

	return row, "", nil
}

func provisionNodes(g *gravity.TestContext, cfg gravity.ProvisionerConfig, param installParam) (gravity.Cluster, error) {
	return g.Provision(cfg.WithOS(param.OSFlavor).
		WithStorageDriver(param.DockerStorageDriver).
		WithNodes(param.NodeCount))
}

func install(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		cluster, err := provisionNodes(g, cfg, param)
		g.OK("VMs ready", err)
		defer func() {
			g.Maybe("uninstall application", g.UninstallApp(cluster.Nodes))
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
	}, nil
}

func provision(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		cluster, err := provisionNodes(g, cfg, param)
		g.OK("provision nodes", err)
		defer func() {
			g.Maybe("destroy", cluster.Destroy())
		}()

		installerURL := cfg.InstallerURL
		if param.InstallerURL != "" {
			installerURL = param.InstallerURL
		}

		g.OK("download installer", g.SetInstaller(cluster.Nodes, installerURL, "install"))
	}, nil
}
