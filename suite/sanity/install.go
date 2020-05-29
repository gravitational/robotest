package sanity

import (
	"encoding/json"
	"time"

	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/robotest/lib/config"

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
	Url     string         `json:"url" validate:"required"`
	Args    []string       `json:"args"`
	Timeout config.Timeout `json:"timeout"`
}

// a hack to set default values when deserializing optional/nil-able structs
// in this case, if the user doesn't specify a script timeout, prefer a non-zero timeout
func (p *scriptParam) UnmarshalJSON(data []byte) error {
	type alias scriptParam // to prevent Unmarshal recursion
	defaults := alias{Timeout: config.Timeout{time.Minute * 5}}
	err := json.Unmarshal(data, &defaults)
	if err != nil {
		return err
	}
	*p = scriptParam(defaults)
	return nil
}

func (p installParam) Save() (row map[string]bigquery.Value, insertID string, err error) {
	row = make(map[string]bigquery.Value)
	row["os"] = p.InstallParam.OSFlavor.Vendor
	row["os_version"] = p.InstallParam.OSFlavor.Version
	row["nodes"] = int(p.NodeCount)
	row["storage"] = p.InstallParam.DockerStorageDriver

	return row, "", nil
}

// withInstallParams returns copy of config applying extended tag to it
func withInstallParam(cfg gravity.ProvisionerConfig, param installParam) gravity.ProvisionerConfig {
	return cfg.
		WithOS(param.OSFlavor).
		WithStorageDriver(param.DockerStorageDriver).
		WithNodes(param.NodeCount)
}

func provisionNodes(g *gravity.TestContext, cfg gravity.ProvisionerConfig, param installParam) (gravity.Cluster, error) {
	return g.Provision(withInstallParam(cfg, param))
}

func install(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		cluster, err := provisionNodes(g, cfg, param)
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
				g.ExecScript(cluster.Nodes, param.Script.Url, param.Script.Args, param.Script.Timeout.Duration))
		}
		g.OK("application installed", g.OfflineInstall(cluster.Nodes, param.InstallParam))
		g.OK("wait for active status", g.WaitForActiveStatus(cluster.Nodes))
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
