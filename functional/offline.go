package functional

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func testOfflineAll(ctx context.Context, t *testing.T, os string, config *Config, tag, dir string) {
	nodes, destroyFn, err := Provision(ctx, t, config, tag, dir, 6, os)
	require.NoError(t, err, "provision nodes")

	destroy := false
	defer func() {
		if destroy {
			destroyFn()
		}
	}()

	ok := t.Run("installOffline 1 node", func(t *testing.T) {
		testOfflineInstall(ctx, t, nodes[0:1])
	})
	require.True(t, ok, "installOffline 1 node")

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
