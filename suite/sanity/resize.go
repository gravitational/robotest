package sanity

import (
	"context"
	"testing"

	"time"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

type resizeParam struct {
	// ReasonableTimeout is per-node operation timeout value
	ReasonableTimeout time.Duration
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
	return func(baseContext context.Context, t *testing.T, baseConfig *gravity.ProvisionerConfig) {
		require.True(t, param.ReasonableTimeout >= minTimeout,
			"timeout value %v too small", param.ReasonableTimeout)

		config := baseConfig.WithNodes(4)

		provisioningCtx, cancelProvision := context.WithTimeout(baseContext, provisioningTimeout)
		defer cancelProvision()
		nodes, destroyFn, err := gravity.Provision(provisioningCtx, t, config)
		require.NoError(t, err, "provision nodes")
		require.Len(t, nodes, 4)
		defer destroyFn(baseContext, t)

		t.Log(nodes)
		ok := t.Run("wipe nodes", func(t *testing.T) {
			t.Skip()
			gravity.Uninstall(provisioningCtx, t, nodes)
		})
		require.True(t, ok, "wipe nodes if re-entrant")

		ok = t.Run("install 1 node", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(baseContext, param.ReasonableTimeout)
			defer cancel()

			err := gravity.OfflineInstall(ctx, t, nodes[0:1], param.InitialFlavor, param.Role)
			require.NoError(t, err)
		})
		require.True(t, ok, "installOffline 1 node")

		ok = t.Run("expand to 3 nodes", func(t *testing.T) {
			expandTo := nodes[1:3]

			ctx, cancel := context.WithTimeout(baseContext, param.ReasonableTimeout*time.Duration(len(expandTo)))
			defer cancel()

			err := gravity.Expand(ctx, t, nodes[0:1], expandTo, param.Role)
			require.NoError(t, err)
		})
		require.True(t, ok, "expand to 3 nodes")

		// TODO: detect current active master and evict it
		ok = t.Run("remove node #0", func(t *testing.T) {
			// FIXME: it is supposed to work without extra time
			time.Sleep(time.Second * 30)

			ctx, cancel := context.WithTimeout(baseContext, param.ReasonableTimeout)
			defer cancel()

			gravity.NodeLoss(ctx, t, nodes[1:3], nodes[0:1])
		})
		require.True(t, ok, "remove one node")

		ok = t.Run("query status from remaining nodes", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(baseContext, statusTimeout)
			defer cancel()

			gravity.Status(ctx, t, nodes[0:1])
			// this is informative, we not assert errors reporting
		})
		require.True(t, ok, "query status after node loss")

		ok = t.Run("recover node", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(baseContext, param.ReasonableTimeout)
			defer cancel()

			err = gravity.Expand(ctx, t, nodes[1:3], nodes[3:4], param.Role)
		})
		require.True(t, ok, "recover node")
	}
}
