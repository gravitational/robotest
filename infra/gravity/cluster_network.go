package gravity

import (
	"context"

	"github.com/gravitational/trace"
)

// Disconnect disconnects the node from the cluster.
func (c *TestContext) Disconnect(node Gravity) error {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.Status)
	defer cancel()
	err := node.Disconnect(ctx)
	return trace.Wrap(err)
}

// Connect connects the node to the cluster.
func (c *TestContext) Connect(node Gravity) error {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.Status)
	defer cancel()
	err := node.Connect(ctx)
	return trace.Wrap(err)
}
