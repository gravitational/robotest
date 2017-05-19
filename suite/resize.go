package suite

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testExpand(ctx context.Context, t *testing.T, current, extra []gravity.Gravity) {
	joinAddr := current[0].Node().PrivateAddr()
	status, err := current[0].Status(ctx)
	require.NoError(t, err, "cluster status")

	logFn := gravity.Logf(t, "testExpand")

	errs := make(chan error)
	for _, node := range extra {
		go func(n gravity.Gravity) {
			errs <- n.Join(ctx, gravity.JoinCmd{
				PeerAddr: joinAddr,
				Token:    status.Token,
				Role:     defaultRole})
		}(node)
	}

	for range extra {
		assert.NoError(t, <-errs, "cluster join")
	}

	all := append([]gravity.Gravity{}, current...)
	all = append(all, extra...)

	for _, node := range all {
		status, err := node.Status(ctx)
		assert.NoError(t, err, "node status")
		logFn("node %s status=%+v", node.Node().Addr(), status)
		// TODO: proper assertion here
	}
}

func testShrink(ctx context.Context, t *testing.T, nodesKeep, nodesRemove []gravity.Gravity) {
	return
}
