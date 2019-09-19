package gravity

import (
	"context"
	"fmt"

	"github.com/gravitational/trace"
)

// Failover isolates the current leader node and elects a new leader node.
func (c *TestContext) Failover(nodes []Gravity) error {
	// TODO: Configure timeouts
	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.Status)
	defer cancel()

	oldLeader, err := getLeaderNode(ctx, nodes)
	if err != nil {
		return trace.Wrap(err)
	}
	c.Logger().Info(fmt.Sprintf("oldLeader=%v", oldLeader))

	if err := oldLeader.Disconnect(ctx); err != nil {
		return trace.Wrap(err, "failed to disconnect from cluster")
	}

	newLeader, err := getLeaderNode(ctx, nodes)
	if err != nil {
		return trace.Wrap(err)
	}
	c.Logger().Info(fmt.Sprintf("newLeader=%v", newLeader))

	if newLeader == oldLeader {
		return trace.BadParameter("did not failover to new leader")
	}

	if err := oldLeader.Connect(ctx); err != nil {
		return trace.Wrap(err, "failed to connect to cluster")
	}

	// TODO: do we need to check the status of all the nodes?
	status, err := newLeader.Status(ctx)
	if err != nil {
		trace.Wrap(err)
	}
	// TODO: add Status.IsActive function
	if status.Cluster.Status != "active" {
		return trace.Wrap(err)
	}

	return nil
}

// getLeaderNode returns the current leader node.
func getLeaderNode(ctx context.Context, nodes []Gravity) (leader Gravity, err error) {
	for _, node := range nodes {
		if node.IsLeader(ctx) {
			if leader != nil {
				return nil, trace.BadParameter("multiple leader nodes")
			}
			leader = node
		}
	}
	if leader == nil {
		return nil, trace.NotFound("unable to get leader node")
	}
	return leader, nil
}
