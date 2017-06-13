package sanity

import (
	"context"
	"fmt"
	"testing"

	"time"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

type resizeParam struct {
	// Timeouts is per-node operation timeout value
	Timeouts gravity.OpTimeouts
	// Role is node role
	Role string
	// InitialFlavor is equivalent to 1 node
	InitialFlavor string
}

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

const (
	// reasonable timeframe a node should report its status including all overheads
	statusTimeout = time.Second * 30
	// reasonable timeframe infrastructure should be provisioned
	provisioningTimeout = time.Minute * 20
	// minTimeout to guard against forgotten params or ridiculously small values
	minTimeout = time.Minute * 5
)

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

// testEssentialResize performs the following sanity test:
// * install 1 node expand to 3
// * force remove 1 recover another one
func basicResize(param resizeParam) gravity.TestFunc {
	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		config := baseConfig.WithNodes(4)

		nodes, destroyFn, err := gravity.Provision(ctx, t, config)
		require.NoError(t, err, "provision nodes")
		defer destroyFn(ctx, t)

		g := gravity.NewContext(ctx, t, param.Timeouts)

		g.OK("install one node", g.OfflineInstall(nodes[0:1], param.InitialFlavor, param.Role))
		g.OK("status", g.Status(nodes[0:1]))

		g.OK("expand to three", g.Expand(nodes[0:1], nodes[1:3], param.Role))
		g.OK("status", g.Status(nodes[0:3]))

		g.OK("node loss", g.NodeLoss(nodes[1:3], nodes[0:1]))
		g.Status(nodes[0:1]) // informative

		g.OK("replace node", g.Expand(nodes[1:3], nodes[3:4], param.Role))
		g.OK("status", g.Status(nodes[1:4]))
	}
}
