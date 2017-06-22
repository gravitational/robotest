package gravity

import (
	"context"

	"github.com/gravitational/robotest/lib/defaults"
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

	master := current[0]
	joinAddr := master.Node().PrivateAddr()
	status, err := master.Status(ctx)
	if err != nil {
		trace.Wrap(err, "query status from [%v]", master)
	}

	ctx, cancel = context.WithTimeout(c.parent, withDuration(c.timeouts.Install, len(extra)))
	defer cancel()

	retry := wait.Retryer{
		Attempts: 1000,
		Delay:    defaults.RetryDelay,
	}
	for _, node := range extra {
		// workaround for bug #2324
		err = retry.Do(ctx, waitEtcdHealthOk(ctx, master))
		if err != nil {
			return trace.Wrap(err, "error waiting for ETCD health on node %s: %v", node.String(), err)
		}

		err = node.Join(ctx, JoinCmd{
			PeerAddr: joinAddr,
			Token:    status.Token,
			Role:     role})
		if err != nil {
			return trace.Wrap(err, "error joining cluster on node %s: %v", node.String(), err)
		}
	}

	return nil
}

func waitEtcdHealthOk(ctx context.Context, node Gravity) func() error {
	return func() error {
		_, exitCode, err := sshutils.RunAndParse(ctx, node,
			`sudo /usr/bin/gravity enter -- --notty /usr/bin/etcdctl -- cluster-health`,
			nil, sshutils.ParseDiscard)
		if err == nil {
			return nil
		}

		if exitCode > 0 {
			return wait.ContinueRetry{err.Error()}
		} else {
			return wait.AbortRetry{err}
		}
	}
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
func (c TestContext) RemoveNode(nodesToKeep []Gravity, remove Gravity) error {
	if len(nodesToKeep) == 0 {
		return trace.BadParameter("node list empty")
	}

	master := nodesToKeep[0]

	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Leave)
	defer cancel()

	err := master.Remove(ctx, remove.Node().PrivateAddr(), !remove.Offline())
	if err != nil {
		return trace.Wrap(err)
	}

	err = c.Status(nodesToKeep)
	return trace.Wrap(err)
}
