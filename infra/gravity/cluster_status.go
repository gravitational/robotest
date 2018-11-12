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
func (c *TestContext) Status(nodes []Gravity) error {
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
				_, err := n.Status(ctx)
				errs <- err
			}(node)
		}

		err := utils.CollectErrors(ctx, errs)
		if err == nil {
			return nil
		}
		c.Logger().Warn("status not available on some nodes, will retry")
		return wait.Continue("status not ready on some nodes")
	})

	return trace.Wrap(err)
}

// CheckTime walks around all nodes and checks whether their time is within acceptable limits
func (c *TestContext) CheckTimeSync(nodes []Gravity) error {
	timeNodes := []sshutils.SshNode{}
	for _, n := range nodes {
		timeNodes = append(timeNodes, sshutils.SshNode{
			Client: n.Client(),
			Log:    c.Logger(),
		})
	}

	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Status)
	defer cancel()

	err := sshutils.CheckTimeSync(ctx, timeNodes)
	return trace.Wrap(err)
}

// PullLogs requests logs from all nodes
// prefix `postmortem` is reserved for cleanup procedure
func (c *TestContext) CollectLogs(prefix string, nodes []Gravity) error {
	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.CollectLogs)
	defer cancel()

	errs := make(chan error, len(nodes))

	c.Logger().WithField("nodes", nodes).Debug("Collecting logs from nodes")
	for _, node := range nodes {
		go func(n Gravity) {
			localPath, err := n.CollectLogs(ctx, prefix)
			if err == nil {
				n.Logger().Debugf("logs in %s", localPath)
			} else {
				n.Logger().WithError(err).Error("error fetching node logs")
			}
			errs <- err
		}(node)
	}

	return trace.Wrap(utils.CollectErrors(ctx, errs))
}

// ClusterNodesByRole defines which roles every node plays in a cluster
type ClusterNodesByRole struct {
	// ApiMaster is Kubernetes apiserver master
	ApiMaster Gravity
	// ClusterMaster is current gravity-site application master
	ClusterMaster Gravity
	// ClusterBackup are backup nodes for gravity-site application
	ClusterBackup []Gravity
	// Regular nodes are those which are part of the cluster but have no role assigned
	Regular []Gravity
}

// NodesByRole will conveniently organize nodes according to their roles in cluster
func (c *TestContext) NodesByRole(nodes []Gravity) (*ClusterNodesByRole, error) {
	const apiserver = "leader.telekube.local"
	if len(nodes) < 1 {
		return nil, trace.BadParameter("at least one node required")
	}

	roles := ClusterNodesByRole{}

	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Status)
	defer cancel()

	apiMaster, err := ResolveInPlanet(ctx, nodes[0], apiserver)
	if err != nil {
		return nil, trace.Wrap(err, "resolving %v: %v", apiserver, err)
	}

	// Run query on the apiserver
	var queryNode Gravity
	for _, node := range nodes {
		if node.Node().PrivateAddr() == apiMaster {
			queryNode = node
			break
		}
	}

	if queryNode == nil {
		return nil, trace.NotFound("failed to find a master node")
	}

	pods, err := KubectlGetPods(ctx, queryNode, kubeSystemNS, appGravityLabel)
	if err != nil {
		return nil, trace.Wrap(err)
	}

nodeLoop:
	for _, node := range nodes {
		ip := node.Node().PrivateAddr()

		if ip == apiMaster {
			roles.ApiMaster = node
		}

		for _, pod := range pods {
			if ip == pod.NodeIP {
				if pod.Ready {
					roles.ClusterMaster = node
				} else {
					roles.ClusterBackup = append(roles.ClusterBackup, node)
				}
				continue nodeLoop
			}
		}

		roles.Regular = append(roles.Regular, node)
	}

	return &roles, nil
}
