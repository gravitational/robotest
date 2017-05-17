package functional

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

const (
	defaultRole = "worker"
)

func testOfflineInstall(ctx context.Context, t *testing.T, nodes []gravity.Gravity) {
	require.True(t, len(nodes) >= 2, "at least 2 nodes")

	master := nodes[0]
	token := "ROBOTEST"
	err := master.Install(ctx, gravity.InstallCmd{
		Token: token,
	})
	require.NoError(t, err, "gravity master installer")

	errs := make(chan error)
	for _, node := range nodes[1:] {
		go func() {
			// TODO: how to properly define node role ?
			errs <- node.Join(ctx, master.Node().PrivateAddr(), token, defaultRole)
		}()
	}

	for _, node := range nodes[1:] {
		err = <-errs
		require.NoError(t, err, "joining cluster")
	}

	// ensure all nodes see each other
	for _, node := range nodes {
		status, err := node.Status(ctx)
		require.NoError(t, err, "node status")
		// TODO: how to properly verify?
	}
}
