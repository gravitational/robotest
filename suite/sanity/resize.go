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
	"fmt"

	"github.com/gravitational/robotest/infra/gravity"

	"cloud.google.com/go/bigquery"
	"github.com/gravitational/trace"
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
		defer func() {
			g.Maybe("destroy", cluster.Destroy())
		}()

		g.OK("download installer", g.SetInstaller(cluster.Nodes, cfg.InstallerURL, "install"))
		g.OK(fmt.Sprintf("install on %d node", param.NodeCount),
			g.OfflineInstall(cluster.Nodes[:param.NodeCount], param.InstallParam))
		g.OK("wait for active status", g.WaitForActiveStatus(cluster.Nodes[:param.NodeCount]))
		g.OK("time sync", g.CheckTimeSync(cluster.Nodes))
		g.OK(fmt.Sprintf("expand to %d nodes", param.ToNodes),
			g.Expand(cluster.Nodes[:param.NodeCount], cluster.Nodes[param.NodeCount:param.ToNodes],
				param.InstallParam))
		g.OK("wait for active status", g.WaitForActiveStatus(cluster.Nodes[:param.ToNodes]))
	}, nil
}
