package sanity

import (
	"fmt"

	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/trace"

	"cloud.google.com/go/bigquery"
)

type resizeParam struct {
	installParam
	// TargetNodes is how many nodes cluster should have after expand
	ToNodes uint `json:"to" validate:"required,gte=3"`
}

func (p resizeParam) Save() (row map[string]bigquery.Value, insertID string, err error) {
	row, _, err = p.installParam.Save()
	if err != nil {
		return nil, "", trace.Wrap(err)
	}

	row["resize_to"] = int(p.ToNodes)
	return row, "", nil
}

// resize installs an initial cluster and then expands or gracefully shrinks it to given number of nodes
func resize(p interface{}) (gravity.TestFunc, error) {
	param := p.(resizeParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		cluster, err := g.Provision(cfg.WithOS(param.OSFlavor).
			WithStorageDriver(param.DockerStorageDriver).
			WithNodes(param.ToNodes))
		g.OK("provision nodes", err)
		defer cluster.Destroy()

		g.OK("download installer", g.SetInstaller(cluster.Nodes, cfg.InstallerURL, "install"))
		g.OK(fmt.Sprintf("install on %d node", param.NodeCount),
			g.OfflineInstall(cluster.Nodes[0:param.NodeCount], param.InstallParam))
		g.OK("status", g.Status(cluster.Nodes[0:param.NodeCount]))
		g.OK("time sync", g.CheckTimeSync(cluster.Nodes))

		g.OK(fmt.Sprintf("expand to %d nodes", param.ToNodes),
			g.Expand(cluster.Nodes[0:param.NodeCount], cluster.Nodes[param.NodeCount:param.ToNodes],
				param.InstallParam))
		g.OK("status", g.Status(cluster.Nodes[0:param.ToNodes]))
	}, nil
}
