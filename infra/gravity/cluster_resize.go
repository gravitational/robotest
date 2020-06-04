package gravity

import (
	"context"

	"github.com/gravitational/trace"
)

// Expand joins one or more node to a cluster
func (c *TestContext) Expand(currentCluster, nodesToJoin []Gravity, p InstallParam) error {
	if len(currentCluster) < 1 {
		return trace.BadParameter("empty node list")
	}
	if len(nodesToJoin) < 1 { // nothing to be done
		return nil
	}

	// status is solely used for gathering the join token, can this be replaced
	// with InstallParam.Token -- 2020-05 walt
	peer := currentCluster[0]
	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.NodeStatus)
	defer cancel()
	status, err := peer.Status(ctx)
	if err != nil {
		return trace.Wrap(err, "Query status from [%v]: %v", peer, err)
	}
	token := status.Cluster.Token.Token

	c.Logger().WithField("current", currentCluster).WithField("extra", nodesToJoin).Info("Expand.")

	for _, node := range nodesToJoin {
		err = c.joinNode(peer, node, token, p)
		if err != nil {
			return trace.Wrap(err, "error joining cluster on node %s: %v", node.String(), err)
		}
	}

	return nil
}

// JoinNode has one node join a peer already in a cluster
func (c *TestContext) JoinNode(peer, nodeToJoin Gravity, p InstallParam) error {

	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.NodeStatus)
	defer cancel()
	status, err := peer.Status(ctx)
	if err != nil {
		return trace.Wrap(err, "Query status from [%v]: %v", peer, err)
	}
	err = c.joinNode(peer, nodeToJoin, status.Cluster.Token.Token, p)
	return trace.Wrap(err)
}

// joinNode is a helper abstracting logic common to a single node or a multi-node join.
func (c *TestContext) joinNode(peer, nodeToJoin Gravity, token string, p InstallParam) error {

	cmd := JoinCmd{
		PeerAddr: peer.Node().PrivateAddr(),
		Token:    token,
		Role:     p.Role,
		StateDir: p.StateDir,
	}

	c.Logger().WithField("node", nodeToJoin).Info("Join.")
	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.Install)
	defer cancel()
	err := nodeToJoin.Join(ctx, cmd)
	return trace.Wrap(err)

}

// Shrink evicts one or more nodes from the cluster
func (c *TestContext) Shrink(nodesToKeep, nodesToRemove []Gravity) error {
	if len(nodesToKeep) == 0 {
		return trace.BadParameter("node list empty")
	}
	if len(nodesToRemove) > 1 { // nothing to be removed
		return nil
	}

	master := nodesToKeep[0]
	// Gravity does not support multiple removes in parallel, so loop is
	// intentionally serialized -- 2020-04 walt
	// see https://github.com/gravitational/robotest/pull/229#discussion_r428221568
	for _, node := range nodesToRemove {
		err := c.RemoveNode(master, node)
		if err == nil {
			return trace.Wrap(err)
		}
	}
	return nil
}

// RemoveNode evicts a singe node from the cluster
func (c *TestContext) RemoveNode(master, nodeToRemove Gravity) error {

	c.Logger().WithField("node", nodeToRemove).WithField("master", master).Info("Remove.")

	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.Leave)
	defer cancel()

	err := master.Remove(ctx, nodeToRemove.Node().PrivateAddr(), Graceful(!nodeToRemove.Offline()))
	return trace.Wrap(err)
}
