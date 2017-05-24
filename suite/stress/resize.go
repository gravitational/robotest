package stress

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

func multiResize(t *testing.T, ctx context.Context, baseConfig *gravity.ProvisionerConfig) {
	config := baseConfig
	config.NodeCount = 6

	nodes, destroyFn, err := gravity.Provision(ctx, t, config)
	require.NoError(t, err, "provision nodes")
	require.Len(t, nodes, 6)

	destroy := false
	defer func() {
		if destroy {
			destroyFn()
		}
	}()

	ok := t.Run("installOffline 3 nodes", func(t *testing.T) {
		testOfflineInstall(ctx, t, nodes[0:3])
	})
	require.True(t, ok, "installOffline 3 node")

	ok = t.Run("expandOffline to 6 nodes", func(t *testing.T) {
		testExpand(ctx, t, nodes[0:1], nodes[1:6])
	})
	require.True(t, ok, "expandOffline to 6 nodes")

	ok = t.Run("shrinkOffline to 3 nodes", func(t *testing.T) {
		testShrink(ctx, t, nodes[0:3], nodes[4:6])
	})
	require.True(t, ok, "shrinkOffline to 3 nodes")

	destroy = true
}
