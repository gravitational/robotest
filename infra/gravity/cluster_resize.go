package gravity

import (
	"context"

	sshutils "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"
)

func (c *TestContext) Expand(current, extra []Gravity, p InstallParam) error {
	if len(current) == 0 || len(extra) == 0 {
		return trace.Errorf("empty node list")
	}
	if c.provisionerCfg.CloudProvider == "ops" {
		return trace.Errorf("not implemented")
	}

	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Status)
	defer cancel()

	master := current[0]
	joinAddr := master.Node().PrivateAddr()
	status, err := master.Status(ctx)
	if err != nil {
		return trace.Wrap(err, "query status from [%v]", master)
	}

	ctx, cancel = context.WithTimeout(c.parent, withDuration(c.timeouts.Install, len(extra)))
	defer cancel()

	for _, node := range extra {
		err = node.Join(ctx, JoinCmd{
			PeerAddr: joinAddr,
			Token:    status.Token,
			Role:     p.Role,
			StateDir: p.StateDir,
		})
		if err != nil {
			return trace.Wrap(err, "error joining cluster on node %s: %v", node.String(), err)
		}
	}

	return nil
}

func waitEtcdHealthOk(ctx context.Context, node Gravity) func() error {
	return func() error {
		exitCode, err := sshutils.RunAndParse(ctx, node.Client(), node.Logger(),
			`sudo /usr/bin/gravity enter -- --notty /usr/bin/etcdctl -- cluster-health`,
			nil, sshutils.ParseDiscard)
		if err == nil {
			return nil
		}

		if exitCode > 0 {
			return wait.Continue(err.Error())
		} else {
			return wait.Abort(err)
		}
	}
}

// ShrinkLeave will gracefully leave cluster
func (c *TestContext) ShrinkLeave(nodesToKeep, nodesToRemove []Gravity) error {
	ctx, cancel := context.WithTimeout(c.parent, withDuration(c.timeouts.Leave, len(nodesToRemove)))
	defer cancel()

	errs := make(chan error, len(nodesToRemove))
	for _, node := range nodesToRemove {
		go func(n Gravity) {
			err := n.Leave(ctx, Graceful(true))
			errs <- trace.Wrap(err, n.Node().PrivateAddr())
		}(node)
	}

	return trace.Wrap(utils.CollectErrors(ctx, errs))
}

// RemoveNode simulates sudden nodes loss within an existing cluster followed by node eviction
func (c *TestContext) RemoveNode(nodesToKeep []Gravity, remove Gravity) error {
	if len(nodesToKeep) == 0 {
		return trace.BadParameter("node list empty")
	}

	master := nodesToKeep[0]

	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Leave)
	defer cancel()

	err := master.Remove(ctx, remove.Node().PrivateAddr(), Graceful(!remove.Offline()))
	if err != nil {
		return trace.Wrap(err)
	}

	err = c.Status(nodesToKeep)
	return trace.Wrap(err)
}
