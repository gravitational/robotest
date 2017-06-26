package sanity

import (
	"context"
	"fmt"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

type expandParam struct {
	// Timeouts is per-node operation timeout value
	Timeouts gravity.OpTimeouts
	// Role is node role
	Role string
	// InitialFlavor is equivalent to 1 node
	InitialFlavor string
	// InitialNodes is how many nodes on first install
	InitialNodes uint
	// TargetNodes is how many nodes cluster should have after expand
	TargetNodes uint
}

// basicExpand installs an initial cluster and then expands it to given number of nodes
func basicExpand(param expandParam) gravity.TestFunc {
	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		config := baseConfig.WithNodes(param.TargetNodes)

		nodes, destroyFn, err := gravity.Provision(ctx, t, config)
		require.NoError(t, err, "provision nodes")
		defer destroyFn(ctx, t)

		g := gravity.NewContext(ctx, t, param.Timeouts)

		g.OK("download installer", g.SetInstaller(nodes, config.InstallerURL, "install"))
		g.OK(fmt.Sprintf("install on %d node", param.InitialNodes),
			g.OfflineInstall(nodes[0:param.InitialNodes], param.InitialFlavor, param.Role))
		g.OK("status", g.Status(nodes[0:param.InitialNodes]))
		g.OK("time sync", g.CheckTimeSync(nodes))

		g.OK(fmt.Sprintf("expand to %d nodes", param.TargetNodes),
			g.Expand(nodes[0:param.InitialNodes], nodes[param.InitialNodes:param.TargetNodes], param.Role))
		g.OK("status", g.Status(nodes[0:param.TargetNodes]))
	}
}
