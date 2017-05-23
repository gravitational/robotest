package sanity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

// testEssentialResize performs the following sanity test:
// * install 1 node expand to 3
// * force remove 1 recover another one
//
func basicResize(ctx context.Context, t *testing.T, baseConfig *gravity.ProvisionerConfig, payload interface{}) {
	config := baseConfig.WithNodes(4)

	nodes, destroyFn, err := gravity.Provision(ctx, t, config)
	require.NoError(t, err, "provision nodes")
	require.Len(t, nodes, 4)
	defer gravity.Destroy(t, destroyFn)

	t.Log(nodes)
	ok := t.Run("wipe nodes", func(t *testing.T) {
		t.Skip()
		gravity.Uninstall(ctx, t, nodes)
	})
	require.True(t, ok, "wipe nodes if re-entrant")

	ok = t.Run("install 1 node", func(t *testing.T) {
		gravity.OfflineInstall(ctx, t, nodes[0:1])
	})
	require.True(t, ok, "installOffline 1 node")

	ok = t.Run("expand to 3 nodes", func(t *testing.T) {
		gravity.Expand(ctx, t, nodes[0:1], nodes[1:3])
	})
	require.True(t, ok, "expand to 3 nodes")

	// TODO: detect current active master and evict it
	ok = t.Run("remove node #0", func(t *testing.T) {
		gravity.NodeLoss(ctx, t, nodes[1], nodes[0:1])
	})
	require.True(t, ok, "remove one node")

	ok = t.Run("recover node", func(t *testing.T) {
		gravity.Expand(ctx, t, nodes[1:3], nodes[3:4])
	})
	require.True(t, ok, "recover node")
}
