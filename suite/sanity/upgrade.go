package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/trace"

	"cloud.google.com/go/bigquery"
)

type upgradeParam struct {
	installParam
	// BaseInstallerURL is initial app installer URL
	// it may be left blank if there's a suitable snapshot to restore
	// based on "version" from installParam
	BaseInstallerURL string `json:"from,omitempty"`
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
		cfg = cfg.WithOS(param.OSFlavor).
			WithStorageDriver(param.DockerStorageDriver).
			WithNodes(param.NodeCount)

		nodes, err := g.RestoreCheckpoint(cfg, checkpointInstall, param.installParam)

		if trace.IsNotFound(err) {
			g.Require("when no snapshot is available, URL to install from is required",
				param.BaseInstallerURL != "")

			nodes, err = provisionNodes(g, cfg, param.provisionParam)
			g.OK("provision nodes", err)

			g.OK("base installer", g.SetInstaller(nodes, param.BaseInstallerURL, "base"))
			g.OK("install", g.OfflineInstall(nodes, param.InstallParam))
			g.OK("status", g.Status(nodes))
		} else {
			g.OK("restoring install from checkpoint", err)
			g.OK("status", g.Status(nodes))
		}

		g.OK("upgrade", g.Upgrade(nodes, cfg.InstallerURL, "upgrade"))
		g.OK("status", g.Status(nodes))
	}, nil
}
