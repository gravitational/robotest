package gravity

import (
	"context"

	"github.com/gravitational/robotest/lib/utils"
)

// Status walks around all nodes and checks whether they all feel OK
func (c TestContext) Status(nodes []Gravity) error {
	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Status)
	defer cancel()

	errs := make(chan error, len(nodes))

	for _, node := range nodes {
		go func(n Gravity) {
			status, err := n.Status(ctx)
			n.Logf("status=%+v", n, status)
			errs <- err
		}(node)
	}

	return utils.CollectErrors(ctx, errs)
}

// SiteReport runs site report command across nodes
func (c TestContext) SiteReport(nodes []Gravity) error {
	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Status)
	defer cancel()

	errs := make(chan error, len(nodes))

	for _, node := range nodes {
		go func(n Gravity) {
			err := n.SiteReport(ctx)
			errs <- err
		}(node)
	}

	return utils.CollectErrors(ctx, errs)
}

// PullLogs requests logs from all nodes
// prefix `postmortem` is reserved for cleanup procedure
func (c TestContext) CollectLogs(prefix string, nodes []Gravity) error {
	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.CollectLogs)
	defer cancel()

	errs := make(chan error, len(nodes))

	c.t.Logf("Collecting logs from nodes %v", nodes)
	for _, node := range nodes {
		go func(n Gravity) {
			localPath, err := n.CollectLogs(ctx, prefix)
			if err == nil {
				n.Logf("logs in %s", localPath)
			} else {
				n.Logf("error fetching logs: %v", err)
			}
			errs <- err
		}(node)
	}

	return utils.CollectErrors(ctx, errs)
}
