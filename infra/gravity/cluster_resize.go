package gravity

import (
	"context"
	"time"

	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"
)

func (c TestContext) Expand(current, extra []Gravity, role string) error {
	if len(current) == 0 || len(extra) == 0 {
		return trace.Errorf("empty node list")
	}

	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Status)
	defer cancel()

	joinAddr := current[0].Node().PrivateAddr()
	status, err := current[0].Status(ctx)
	if err != nil {
		trace.Wrap(err, "query status from [%v]", current[0])
	}

	ctx, cancel = context.WithTimeout(c.parent, withDuration(c.timeouts.Install, len(extra)))
	defer cancel()

	errs := make(chan error, len(extra))
	for _, node := range extra {
		go func(n Gravity) {
			err := n.Join(ctx, JoinCmd{
				PeerAddr: joinAddr,
				Token:    status.Token,
				Role:     role})
			errs <- trace.Wrap(err, n.Node().PrivateAddr())
		}(node)
		time.Sleep(time.Second)
	}

	_, err = utils.Collect(ctx, cancel, errs, nil)
	return trace.Wrap(err)
	// TODO: make proper assertion
}

// ShrinkLeave will gracefully leave cluster
func (c TestContext) ShrinkLeave(nodesToKeep, nodesToRemove []Gravity) error {
	ctx, cancel := context.WithTimeout(c.parent, withDuration(c.timeouts.Leave, len(nodesToRemove)))
	defer cancel()

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
func (c TestContext) NodeLoss(nodesToKeep []Gravity, nodesToRemove []Gravity) error {
	if len(nodesToKeep) == 0 || len(nodesToRemove) == 0 {
		return trace.BadParameter("node list empty")
	}

	ctx, cancel := context.WithTimeout(c.parent, time.Minute)
	defer cancel()

	errs := make(chan error, len(nodesToRemove))
	for _, node := range nodesToRemove {
		go func(n Gravity) {
			err := n.PowerOff(ctx, Force) // force
			errs <- trace.Wrap(err, n.Node().PrivateAddr())
		}(node)
	}

	if err := utils.CollectErrors(ctx, errs); err != nil {
		return trace.Wrap(err, "error powering off")
	}

	// informative, is expected to report errors
	c.Status(nodesToKeep)

	master := nodesToKeep[0]
	evictErrors := []error{}

	ctx, cancel = context.WithTimeout(c.parent, withDuration(c.timeouts.Leave, len(nodesToRemove)))
	defer cancel()

	for _, node := range nodesToRemove {
		evictErrors = append(evictErrors, wait.Retry(ctx, tryEvict(ctx, master, node)))
	}
	return trace.NewAggregate(evictErrors...)
}

func tryEvict(ctx context.Context, master, node Gravity) func() error {
	return func() error {
		// it seems it may fail sometimes, so we will repeat
		err := master.Remove(ctx, node.Node().PrivateAddr(), Force)
		if err != nil {
			return wait.ContinueRetry{err.Error()}
		}
		return nil
	}
}
