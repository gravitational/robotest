/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sanity

import (
	"time"

	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/robotest/lib/config"
	"github.com/gravitational/trace"

	"cloud.google.com/go/bigquery"
)

type installParam struct {
	gravity.InstallParam
	// NodeCount is how many nodes
	NodeCount uint `json:"nodes" validate:"gte=1"`
	// Script if not empty would be executed with args provided after installer has been transferred
	Script *scriptParam `json:"script"`
}

func (r *installParam) CheckAndSetDefaults() error {
	if r.Script != nil {
		if err := r.Script.CheckAndSetDefaults(); err != nil {
			return trace.Wrap(err)
		}
	}
	// TODO assert NodeCount >= flavor
	return nil
}

type scriptParam struct {
	Url     string          `json:"url" validate:"required"`
	Args    []string        `json:"args"`
	Timeout *config.Timeout `json:"timeout" validate:"omitempty"`
}

func (r *scriptParam) CheckAndSetDefaults() error {
	if r.Timeout == nil {
		r.Timeout = &config.Timeout{Duration: 5 * time.Minute}
	} else {
		if r.Timeout.Duration < 0 {
			return trace.BadParameter("timeout must be >= 0")
		}
	}
	if r.Url == "" {
		return trace.BadParameter("script URL must be specified")
	}
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
