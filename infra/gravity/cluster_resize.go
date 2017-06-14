package gravity

import (
	"context"
	"time"

	sshutils "github.com/gravitational/robotest/lib/ssh"
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

	for i, node := range extra {
		if i > 0 {
			err = wait.Retry(ctx, waitEtcdHealthOk(ctx, node))
			if err != nil {
				return trace.Wrap(err, "error waiting for ETCD health on node %s: %v", node.String(), err)
			}
		}

		err = n.Join(ctx, JoinCmd{
			PeerAddr: joinAddr,
			Token:    status.Token,
			Role:     role})
		if err != nil {
			return trace.Wrap(err, "error joining cluster on node %s: %v", node.String(), err)
		}
	}

	return nil
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

func waitEtcdHealthOk(ctx context.Context, node Gravity) func() error {
	return func() error {
		_, exitCode, err := sshutils.RunAndParse(ctx, node,
			"cd %s && sudo ./gravity enter -- /usr/bin/etcdctl -- cluster-health", nil)
		if err == nil {
			return nil
		}

		if exitCode > 0 {
			return wait.ContinueRetry{err.Error()}
		} else {
			return wait.AbortRetry{err.Error()}
		}
	}
}
