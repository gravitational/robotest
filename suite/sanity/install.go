package sanity

import (
	"time"

	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/trace"

	"cloud.google.com/go/bigquery"
)

type installParam struct {
	gravity.InstallParam
	// NodeCount is how many nodes
	NodeCount uint `json:"nodes" validate:"gte=1"`
}

func (p installParam) Save() (row map[string]bigquery.Value, insertID string, err error) {
	row = make(map[string]bigquery.Value)
	row["os"] = p.InstallParam.OSFlavor.Vendor
	row["os_version"] = p.InstallParam.OSFlavor.Version
	row["nodes"] = int(p.NodeCount)
	row["storage"] = p.InstallParam.DockerStorageDriver

	return row, "", nil
}

func provisionNodes(g *gravity.TestContext, cfg gravity.ProvisionerConfig, param installParam) (nodes []gravity.Gravity, err error) {
	cfg = cfg.WithOS(param.OSFlavor).
		WithStorageDriver(param.DockerStorageDriver).
		WithNodes(param.NodeCount)
	nodes, err = g.RestoreCheckpoint(cfg, checkpointInstall, param)

	if err == nil {
		return nodes, err
	}

	if !trace.IsNotFound(err) {
		return nil, trace.Wrap(err)
	}

	nodes, err = g.Provision(cfg)
	return nodes, trace.Wrap(err)
}

func install(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		nodes, err := provisionNodes(g, cfg, param)
		g.OK("VMs ready", err)

		g.OK("installer downloaded", g.SetInstaller(nodes, cfg.InstallerURL, "install"))
		g.OK("application installed", g.OfflineInstall(nodes, param.InstallParam))
		g.OK("status", g.Status(nodes))
	}, nil
}

func provision(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		_, err := provisionNodes(g, cfg, param)
		g.OK("provision nodes", err)
		time.Sleep(time.Minute * 5)

		//g.OK("download installer", g.SetInstaller(nodes, cfg.InstallerURL, "install"))
	}, nil
}
