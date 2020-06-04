package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"

	"cloud.google.com/go/bigquery"
	"github.com/gravitational/trace"
)

type upgradeParam struct {
	installParam
	// BaseInstallerURL is initial app installer URL
	BaseInstallerURL string `json:"from" validate:"required"`
}

func (p upgradeParam) Save() (row map[string]bigquery.Value, insertID string, err error) {
	row, _, err = p.installParam.Save()
	if err != nil {
		return nil, "", trace.Wrap(err)
	}

	row["upgrade_from"] = p.BaseInstallerURL
	return row, "", nil
}

func upgrade(p interface{}) (gravity.TestFunc, error) {
	param := p.(upgradeParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		cluster, err := provisionNodes(g, cfg, param.installParam)
		g.OK("provision nodes", err)
		defer func() {
			g.Maybe("destroy", cluster.Destroy())
		}()

		g.OK("base installer", g.SetInstaller(cluster.Nodes, param.BaseInstallerURL, "base"))
		g.OK("install", g.OfflineInstall(cluster.Nodes, param.InstallParam))
		g.OK("wait for active status", g.WaitForActiveStatus(cluster.Nodes))
		g.OK("upgrade", g.Upgrade(cluster.Nodes, cfg.InstallerURL, cfg.GravityURL, "upgrade"))
		g.OK("wait for active status", g.WaitForActiveStatus(cluster.Nodes))
	}, nil
}
