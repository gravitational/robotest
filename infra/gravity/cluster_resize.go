package gravity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"
)

func Expand(ctx context.Context, t *testing.T, current, extra []Gravity, role string) error {
	if len(current) == 0 || len(extra) == 0 {
		return trace.Errorf("empty node list")
	}

	joinAddr := current[0].Node().PrivateAddr()
	status, err := current[0].Status(ctx)
	if err != nil {
		trace.Wrap(err, "query status from [%v]", current[0])
	}

	errs := make(chan error, len(extra))
	for _, node := range extra {
		go func(n Gravity) {
			err := n.Join(ctx, JoinCmd{
				PeerAddr: joinAddr,
				Token:    status.Token,
				Role:     role})
			errs <- trace.Wrap(err, n.Node().PrivateAddr())
		}(node)
	}

	err = utils.CollectErrors(ctx, errs)
	if err != nil {
		return trace.Wrap(err)
	}

	all := append([]Gravity{}, current...)
	all = append(all, extra...)

	return trace.Wrap(Status(ctx, t, all))
}

// ShrinkLeave will gracefully leave cluster
func ShrinkLeave(ctx context.Context, t *testing.T, nodesToKeep, nodesToRemove []Gravity) error {
	errs := make(chan error, len(nodesToRemove))
	for _, node := range nodesToRemove {
		go func(n Gravity) {
			err := n.Leave(ctx, Graceful)
			errs <- trace.Wrap(err, n.Node().PrivateAddr())
		}(node)
	}

	return trace.Wrap(utils.CollectErrors(ctx, errs))
}

// TestNodeLoss simulates sudden nodes loss within an existing cluster followed by node eviction
func NodeLoss(ctx context.Context, t *testing.T, nodesToKeep []Gravity, nodesToRemove []Gravity) error {
	if len(nodesToKeep) == 0 || len(nodesToRemove) == 0 {
		return trace.BadParameter("node list empty")
	}

	errs := make(chan error, len(nodesToRemove))
	for _, node := range nodesToRemove {
		go func(n Gravity) {
			err := n.PowerOff(ctx, Force) // force
			errs <- trace.Wrap(err, n.Node().PrivateAddr())
		}(node)
	}

	err := utils.CollectErrors(ctx, errs)
	if err != nil {
		return trace.Wrap(err, "error powering off")
	}

	// informative, is expected to report errors
	Status(ctx, t, nodesToKeep)

	master := nodesToKeep[0]
	evictErrors := []error{}

	for _, node := range nodesToRemove {
		evictErrors = append(evictErrors, wait.Retry(ctx, tryEvict(ctx, t, master, node)))
	}
	return trace.NewAggregate(evictErrors...)
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
