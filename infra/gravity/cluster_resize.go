package gravity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/robotest/lib/wait"

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

	err = utils.CollectErrors(ctx, errs)
	assert.NoError(t, err, "expand")

	all := append([]Gravity{}, current...)
	all = append(all, extra...)

	err = Status(ctx, t, all)
	assert.NoError(t, err, "all node status after expand")
}

// ShrinkLeave will gracefully leave cluster
func ShrinkLeave(ctx context.Context, t *testing.T, nodesToKeep, nodesToRemove []Gravity) {
	errs := make(chan error, len(nodesToRemove))
	for _, node := range nodesToRemove {
		go func(n Gravity) {
			err := n.Leave(ctx, Graceful)
			errs <- trace.Wrap(err, n.Node().PrivateAddr())
		}(node)
	}

	err := utils.CollectErrors(ctx, errs)
	assert.NoError(t, err, "node leave")

	for _, node := range nodesToKeep {
		status, err := node.Status(ctx)
		assert.NoError(t, err, "node status")
		t.Logf("node %s status=%+v", node.Node().Addr(), status)
		// TODO: proper assertion here
	}
}

// TestNodeLoss simulates sudden nodes loss within an existing cluster followed by node eviction
func NodeLoss(ctx context.Context, t *testing.T, nodesKeep []Gravity, nodesToRemove []Gravity) {
	require.NotZero(t, len(nodesKeep), "need to keep some nodes")

	errs := make(chan error, len(nodesToRemove))
	for _, node := range nodesToRemove {
		go func(n Gravity) {
			err := n.PowerOff(ctx, Force) // force
			errs <- trace.Wrap(err, n.Node().PrivateAddr())
		}(node)
	}

	err := utils.CollectErrors(ctx, errs)
	require.NoError(t, err, "kill nodes")

	Status(ctx, t, nodesKeep)

	master := nodesKeep[0]
	evictErrors := []error{}

	for _, node := range nodesToRemove {
		evictErrors = append(evictErrors, wait.Retry(ctx, tryEvict(ctx, t, master, node)))
	}
	assert.NoError(t, trace.NewAggregate(evictErrors...), "node removal")
}

func tryEvict(ctx context.Context, t *testing.T, master, node Gravity) func() error {
	return func() error {
		// it seems it may fail sometimes, so we will repeat
		err := master.Remove(ctx, node.Node().PrivateAddr(), Force)
		if err != nil {
			t.Logf("(%s) error evicting %s : %v", master, node, err)
			return wait.ContinueRetry{err.Error()}
		}
		return nil
	}
}
