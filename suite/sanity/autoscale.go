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
