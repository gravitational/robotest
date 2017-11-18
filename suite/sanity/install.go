package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/trace"

	"cloud.google.com/go/bigquery"

	log "github.com/sirupsen/logrus"
)

type provisionParam struct {
	gravity.InstallParam
	// NodeCount is how many nodes
	NodeCount uint `json:"nodes" validate:"gte=1"`
}

type installParam struct {
	provisionParam
	Version string `json:"version" validate:"required"`
}

func (p installParam) Save() (row map[string]bigquery.Value, insertID string, err error) {
	row = make(map[string]bigquery.Value)
	row["os"] = p.InstallParam.OSFlavor.Vendor
	row["os_version"] = p.InstallParam.OSFlavor.Version
	row["nodes"] = int(p.NodeCount)
	row["storage"] = p.InstallParam.DockerStorageDriver

	return row, "", nil
}

func provisionNodes(g *gravity.TestContext, cfg gravity.ProvisionerConfig, param provisionParam) (nodes []gravity.Gravity, err error) {
	cfg = cfg.WithOS(param.OSFlavor).
		WithStorageDriver(param.DockerStorageDriver).
		WithNodes(param.NodeCount)
	nodes, err = g.RestoreCheckpoint(cfg, checkpointProvision, param)

	if err == nil {
		return nodes, err
	}

	if trace.IsNotFound(err) {
		g.Logger().Warnf("no VM snapshot found for %+v", param)
	} else {
		g.Logger().WithFields(log.Fields{
			"err":   err,
			"param": param,
		}).Error("failed to provision VM from snapshot - proceeding with creating from scratch")
	}

	nodes, err = g.Provision(cfg)
	return nodes, trace.Wrap(err)
}

func install(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		g.AssertCheckpointDuplicate(cfg.CloudProvider, checkpointInstall, param)

		nodes, err := provisionNodes(g, cfg, param.provisionParam)
		g.OK("VMs ready", err)

		g.OK("installer downloaded", g.SetInstaller(nodes, cfg.InstallerURL, "install"))
		g.OK("application installed", g.OfflineInstall(nodes, param.InstallParam))
		g.Checkpoint(checkpointInstall, nodes)
		g.OK("status", g.Status(nodes))
	}, nil
}

func provision(p interface{}) (gravity.TestFunc, error) {
	param := p.(provisionParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		g.AssertCheckpointDuplicate(cfg.CloudProvider, checkpointProvision, param)

		nodes, err := provisionNodes(g, cfg, param)
		g.OK("provision nodes", err)
		g.Checkpoint(checkpointProvision, nodes)

		g.OK("download installer", g.SetInstaller(nodes, cfg.InstallerURL, "install"))
	}, nil
}
