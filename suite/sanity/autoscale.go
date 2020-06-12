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
)

// autoscale installs an initial cluster and then expands or gracefully shrinks it to given number of nodes
func autoscale(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		cluster, err := provisionNodes(g, cfg, param)
		g.OK("VMs ready", err)
		defer func() {
			g.Maybe("destroy", cluster.Destroy())
		}()

		g.OK("wait for active status on masters", g.WaitForActiveStatus(cluster.Nodes))
		g.OK("time sync", g.CheckTimeSync(cluster.Nodes))

		// Scale Up
		workers, err := g.AutoScale(3)
		g.OK("asg-up", err)
		g.OK("wait for active status on masters", g.WaitForActiveStatus(cluster.Nodes))
		g.OK("wait for active status on asg workers", g.WaitForActiveStatus(workers))

		// Scale Down
		workers, err = g.AutoScale(1)
		g.OK("asg-down", err)
		g.OK("wait for active status on masters", g.WaitForActiveStatus(cluster.Nodes))
		_, err = g.Status(workers)
		g.OK("status on asg workers", err)
	}, nil
}
