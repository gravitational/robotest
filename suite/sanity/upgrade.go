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
	"github.com/gravitational/robotest/infra/gravity"

	"cloud.google.com/go/bigquery"
	"github.com/gravitational/trace"
)

type upgradeParam struct {
	installParam
	// BaseInstallerURL is initial app installer URL
	BaseInstallerURL string `json:"from" validate:"required"`
	GravityURL       string `json:"gravity_url"`
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
		g.OK("upgrade", g.Upgrade(cluster.Nodes, param.InstallerURL, param.GravityURL, "upgrade"))
		g.OK("wait for active status", g.WaitForActiveStatus(cluster.Nodes))
	}, nil
}
