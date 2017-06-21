package gravity

import (
	"context"
	"time"

	sshutils "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"
)

// Status walks around all nodes and checks whether they all feel OK
func (c TestContext) Status(nodes []Gravity) error {
	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Status)
	defer cancel()

	retry := wait.Retryer{
		Attempts: 1000,
		Delay:    time.Second * 20,
	}

	err := retry.Do(ctx, func() error {
		errs := make(chan error, len(nodes))

		for _, node := range nodes {
			go func(n Gravity) {
				status, err := n.Status(ctx)
				n.Logf("status=%+v, err=%v", status, err)
				errs <- err
			}(node)
		}

		err := utils.CollectErrors(ctx, errs)
		if err == nil {
			return nil
		}
		c.Logf("status not available on some nodes, will retry")
		return wait.Continue("status not ready on some nodes")
	})

	return trace.Wrap(err)
}

// CheckTime walks around all nodes and checks whether their time is within acceptable limits
func (c TestContext) CheckTimeSync(nodes []Gravity) error {
	timeNodes := []sshutils.SshNode{}
	for _, n := range timeNodes {
		timeNodes = append(timeNodes, n)
	}

	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Status)
	defer cancel()

	err := sshutils.CheckTimeSync(ctx, timeNodes)

	return trace.Wrap(err)
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

	return trace.Wrap(utils.CollectErrors(ctx, errs))
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

	return trace.Wrap(utils.CollectErrors(ctx, errs))
}

type ClusterNodesByRole struct {
	ApiMaster         Gravity
	GravitySiteMaster Gravity
	GravitySiteBackup []Gravity
	Regular           []Gravity
}

// NodesByRole will conveniently organize nodes according to their roles in cluster
func (c TestContext) NodesByRole(nodes []Gravity) (*ClusterNodesByRole, error) {
	if len(nodes) < 1 {
		return nil, trace.BadParameter("at least one node required")
	}

	roles := ClusterNodesByRole{
		Regular:           []Gravity{},
		GravitySiteBackup: []Gravity{},
	}

	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Status)
	defer cancel()

	apiMasterIP, err := ResolveInPlanet(ctx, nodes[0], "apiserver")
	if err != nil {
		return nil, trace.Wrap(err, "resolving apiserver: %v", err)
	}

	gravityMasterIP, gravityOther, err := GetGravitySiteNodes(ctx, nodes[0])
	if err != nil {
		return nil, trace.Wrap(err)
	}

node_loop:
	for _, node := range nodes {
		if node.Node().PrivateAddr() == apiMasterIP {
			roles.ApiMaster = node
		}

		if node.Node().PrivateAddr() == gravityMasterIP {
			roles.GravitySiteMaster = node
			continue node_loop
		}

		for _, ip := range gravityOther {
			if node.Node().PrivateAddr() == ip {
				roles.GravitySiteBackup = append(roles.GravitySiteBackup, node)
				continue node_loop
			}
		}

		roles.Regular = append(roles.Regular, node)
	}

	return &roles, nil
}
