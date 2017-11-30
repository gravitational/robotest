package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"
)

// autoscale installs an initial cluster and then expands or gracefully shrinks it to given number of nodes
func autoscale(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		masters, destroyFn, err := provisionNodes(g, cfg, param)
		g.OK("VMs ready", err)
		defer destroyFn()

		g.OK("status", g.Status(masters))
		g.OK("time sync", g.CheckTimeSync(masters))

		// Scale Up
		workers, err := g.AutoScale(3)
		g.OK("asg-up", err)
		g.OK("status-masters", g.Status(masters))
		g.OK("status-workers", g.Status(workers))

		// Scale Down
		workers, err = g.AutoScale(1)
		g.OK("asg-down", err)
		g.OK("status-masters", g.Status(masters))
		g.OK("status-workers", g.Status(workers))
	}, nil
}
