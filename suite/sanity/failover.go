/*
Copyright 2018 Gravitational, Inc.

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
)

// TODO
type failoverParam struct {
	installParam
}

// failover tests master failover
func failover(p interface{}) (gravity.TestFunc, error) {
	param := p.(failoverParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		cluster, err := provisionNodes(g, cfg, param.installParam)
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
				g.ExecScript(cluster.Nodes, param.Script.Url, param.Script.Args))
		}
		g.OK("application installed", g.OfflineInstall(cluster.Nodes, param.InstallParam))
		g.OK("status", g.Status(cluster.Nodes))
		g.OK("master failover", g.Failover(cluster.Nodes))
	}, nil
}
