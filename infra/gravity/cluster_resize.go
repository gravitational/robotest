package gravity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/lib/utils"

	"github.com/gravitational/trace"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Expand(ctx context.Context, t *testing.T, current, extra []Gravity) {
	joinAddr := current[0].Node().PrivateAddr()
	status, err := current[0].Status(ctx)
	require.NoError(t, err, "cluster status")

	errs := make(chan error, len(extra))
	for _, node := range extra {
		go func(n Gravity) {
			err := n.Join(ctx, JoinCmd{
				PeerAddr: joinAddr,
				Token:    status.Token,
				Role:     defaultRole})
			errs <- trace.Wrap(err, n.Node().PrivateAddr())
		}(node)
	}

	err = utils.CollectErrors(ctx, len(extra), errs)
	assert.NoError(t, err, "expand")

	all := append([]Gravity{}, current...)
	all = append(all, extra...)

	err = Status(ctx, t, all)
	assert.NoError(t, err, "all node status after expand")
}

// ShrinkLeave will gracefully leave cluster
func ShrinkLeave(ctx context.Context, t *testing.T, nodesKeep, nodesToRemove []Gravity) {
	errs := make(chan error, len(nodesToRemove))
	for _, node := range nodesToRemove {
		go func(n Gravity) {
			err := n.Leave(ctx, Graceful)
			errs <- trace.Wrap(err, n.Node().PrivateAddr())
		}(node)
	}

	err := utils.CollectErrors(ctx, len(nodesToRemove), errs)
	assert.NoError(t, err, "node leave")

	for _, node := range nodesKeep {
		status, err := node.Status(ctx)
		assert.NoError(t, err, "node status")
		t.Logf("node %s status=%+v", node.Node().Addr(), status)
		// TODO: proper assertion here
	}
}

// TestNodeLoss simulates sudden nodes loss within an existing cluster
func NodeLoss(ctx context.Context, t *testing.T, master Gravity, nodesToRemove []Gravity) {
	errs := make(chan error, len(nodesToRemove))
	for _, node := range nodesToRemove {
		go func(n Gravity) {
			err := n.PowerOff(ctx, Force) // force
			errs <- trace.Wrap(err, n.Node().PrivateAddr())
		}(node)
	}

	err := utils.CollectErrors(ctx, len(nodesToRemove), errs)
	assert.NoError(t, err, "kill nodes")

	evictErrors := []error{}
	for _, node := range nodesToRemove {
		err := master.Remove(ctx, node.Node().PrivateAddr(), Force)
		if err != nil {
			evictErrors = append(evictErrors, err)
		}
	}
	assert.NoError(t, trace.NewAggregate(evictErrors...), "node removal")
}
