package functional

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

func testExpand(ctx context.Context, t *testing.T, current, extra []gravity.Gravity) {
	joinAddr := current[0].Node().PrivateAddr()
	status, err := current[0].Status(ctx)
	require.NoError(t, err, "cluster status")

	errs := make(chan error)
	for _, node := range extra {
		go func() {
			errs <- node.Join(ctx, joinAddr, status.Token, defaultRole)
		}()
	}

	for _, node := range extra {
		require.NoError(t, <-errs, "cluster join")
	}

	all := append([]gravity.Gravity{}, current)
	all = append(all, extra)

	for _, node := range all {
		status, err := node.Status(ctx)
		require.NoError(t, err, "node status")
		// TODO: proper assertion here
	}
}

func testShrink(ctx context.Context, t *testing.T, nodesKeep, nodesRemove []gravity.Gravity) {
	t.Skip()
}
