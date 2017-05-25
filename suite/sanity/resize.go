package sanity

import (
	"context"
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

const (
	// reasonable timeframe a node should report its status including all overheads
	statusTimeout = time.Second * 30
	// reasonable timeframe infrastructure should be provisioned
	provisioningTimeout = time.Minute * 20
	// minTimeout to guard against forgotten params or ridiculously small values
	minTimeout = time.Minute * 5
)

// testEssentialResize performs the following sanity test:
// * install 1 node expand to 3
// * force remove 1 recover another one
func basicResize(param resizeParam) gravity.TestFunc {
	return func(ctx context.Context, t *testing.T, baseConfig *gravity.ProvisionerConfig) {
		config := baseConfig.WithNodes(4)

		nodes, destroyFn, err := gravity.Provision(ctx, t, config)
		require.NoError(t, err, "provision nodes")
		defer destroyFn(ctx, t)

		g := gravity.NewContext(ctx, t, param.Timeouts)

		g.OK("install one node", g.OfflineInstall(nodes[0:1], param.InitialFlavor, param.Role))
		g.OK("expand to three", g.Expand(nodes[0:1], nodes[1:3], param.Role))
		g.OK("node loss", g.NodeLoss(nodes[1:3], nodes[0:1]))
		g.Status(nodes[0:1]) // informative
		g.OK("replace node", g.Expand(nodes[1:3], nodes[3:4], param.Role))
	}
}
