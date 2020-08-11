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

	"github.com/sirupsen/logrus"
)

func shrink(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {

		cfg = withInstallParam(cfg, param).WithNodes(param.NodeCount + 1)

		cluster, err := g.Provision(cfg)
		g.OK("Provision nodes.", err)
		defer func() {
			g.Maybe("Destroy.", cluster.Destroy())
		}()

		all := cluster.Nodes
		target := make([]gravity.Gravity, 1)
		copy(target, cluster.Nodes[0:1])
		others := make([]gravity.Gravity, len(all)-1)
		copy(others, cluster.Nodes[1:])
		g.Logger().WithFields(logrus.Fields{"target": target, "others": others}).Info("Select join/shrink target.")

		g.OK("Download installer.", g.SetInstaller(all, param.InstallerURL, "install"))

		g.OK("Install.", g.OfflineInstall(others, param.InstallParam))
		g.OK("Wait for active status.", g.WaitForActiveStatus(others))

		joinParam := param.InstallParam
		joinParam.Role = "knode"
		g.OK("Expand.", g.Expand(others, target, joinParam))
		g.OK("Wait for active status.", g.WaitForActiveStatus(all))

		roles, err := g.NodesByRole(all)
		g.OK("Query roles.", err)
		g.Logger().WithFields(logrus.Fields{"roles": roles, "nodes": all}).Info("Node roles after expand.")

		g.OK("Shrink.", g.Shrink(others, target))
		g.OK("Wait for active status.", g.WaitForActiveStatus(others))

	}, nil
}
