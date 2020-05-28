package gravity

import (
	"context"

	"github.com/cenkalti/backoff"

	"github.com/gravitational/robotest/lib/constants"
	sshutils "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/robotest/lib/wait"
	"github.com/gravitational/trace"
)

// statusValidator returns nil if the Gravity Status is the expected status or an error otherwise.
type statusValidator func(s GravityStatus) error

// checkNotDegraded returns an error if the cluster status is Degraded.
//
// This function is a reimplementation of the logic in https://github.com/gravitational/gravity/blob/7.0.0/lib/status/status.go#L180-L185
func checkNotDegraded(s GravityStatus) error {
	if s.Cluster.State == constants.ClusterStateDegraded {
		return trace.CompareFailed("cluster state %q", s.Cluster.State)
	}
	if s.Cluster.SystemStatus != constants.SystemStatus_Running {
		return trace.CompareFailed("expected system_status %v, found %v", constants.SystemStatus_Running, s.Cluster.SystemStatus)
	}
	return nil
}

// checkActive returns an error if the cluster is degraded or state != active.
func checkActive(s GravityStatus) error {
	if err := checkNotDegraded(s); err != nil {
		return trace.Wrap(err)
	}
	if s.Cluster.State != constants.ClusterStateActive {
		return trace.CompareFailed("expected state %q, found %q", constants.ClusterStateActive, s.Cluster.State)
	}
	return nil
}

// WaitForActiveStatus blocks until all nodes report state = Active and notDegraded or an internal timeout expires.
func (c *TestContext) WaitForActiveStatus(nodes []Gravity) error {
	c.Logger().WithField("nodes", Nodes(nodes)).Info("Waiting for active status.")
	return c.WaitForStatus(nodes, checkActive)
}

// WaitForStatus blocks until all nodes satisfy the expected statusValidator or an internal timeout expires.
func (c *TestContext) WaitForStatus(nodes []Gravity, expected statusValidator) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = c.timeouts.ClusterStatus

	expectStatus := func() (err error) {
		statuses, err := c.Status(nodes)
		if err != nil {
			return trace.Wrap(err)
		}
		for _, status := range statuses {
			err = expected(status)
			if err != nil {
				c.Logger().WithError(err).WithField("status", status).Warn("Unexpected Status.")
				return trace.Wrap(err)
			}
		}
		return nil
	}

	err := wait.RetryWithInterval(c.ctx, b, expectStatus, c.Logger())

	return trace.Wrap(err)

}

// Status queries `gravity status` on each node in nodes.
func (c *TestContext) Status(nodes []Gravity) ([]GravityStatus, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.NodeStatus)
	defer cancel()

	errC := make(chan error, len(nodes))
	valueC := make(chan interface{}, len(nodes))

	for _, node := range nodes {
		go func(n Gravity) {
			status, err := n.Status(ctx)
			errC <- err
			if status != nil {
				valueC <- *status
			}
		}(node)
	}

	values, err := utils.Collect(ctx, cancel, errC, valueC)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	var statuses []GravityStatus
	for _, v := range values {
		status, ok := v.(GravityStatus)
		if !ok {
			return nil, trace.BadParameter("expected %T, got %T", status, v)
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
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

	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.TimeSync)
	defer cancel()
	err := sshutils.CheckTimeSync(ctx, timeNodes)
	return trace.Wrap(err)
}

// CollectLogs requests logs from all nodes.
// prefix `postmortem` is reserved for cleanup procedure
func (c *TestContext) CollectLogs(prefix string, nodes []Gravity) error {
	if len(nodes) < 1 {
		return nil
	}
	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.CollectLogs)
	defer cancel()

	c.Logger().WithField("nodes", nodes).Debug("Collecting logs from nodes.")

	api, other, err := apiserverNode(ctx, nodes)
	if err != nil || api == nil {
		c.Logger().WithError(err).Warn("Unable to determine api-server.")
	}

	errors := make(chan error, len(nodes))
	var args []string

	if api != nil {
		nodes = other // exclude api server from regular collection, as it is handled in this block
		args = []string{"--filter=system", "--filter=kubernetes"}
		go collectLogsFromNode(ctx, api, prefix, args, errors)
	}

	args = []string{"--filter=system"}
	for _, node := range nodes {
		go collectLogsFromNode(ctx, node, prefix, args, errors)
	}

	err = utils.CollectErrors(ctx, errors)

	return trace.Wrap(err)
}

func collectLogsFromNode(ctx context.Context, node Gravity, prefix string, args []string, errors chan<- error) {
	node.Logger().Debug("Fetching node logs.")
	localPath, err := node.CollectLogs(ctx, prefix, args...)
	if err != nil {
		node.Logger().WithError(err).Error("Log fetch failed.")
	} else {
		node.Logger().WithField("path", localPath).Info("Logs saved.")
	}
	errors <- err
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
	// Other lists all nodes but the API server node
	Other []Gravity
}

// NodesByRole will conveniently organize nodes according to their roles in cluster
func (c *TestContext) NodesByRole(nodes []Gravity) (roles *ClusterNodesByRole, err error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.ResolveInPlanet)
	defer cancel()

	roles = &ClusterNodesByRole{}
	roles.ApiMaster, roles.Other, err = apiserverNode(ctx, nodes)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	ctx, cancel = context.WithTimeout(c.ctx, c.timeouts.GetPods)
	defer cancel()
	// Run query on the apiserver
	pods, err := KubectlGetPods(ctx, roles.ApiMaster, kubeSystemNS, appGravityLabel)
	if err != nil {
		return nil, trace.Wrap(err)
	}

L:
	for _, node := range nodes {
		ip := node.Node().PrivateAddr()

		for _, pod := range pods {
			if ip == pod.NodeIP {
				if pod.Ready {
					roles.ClusterMaster = node
				} else {
					roles.ClusterBackup = append(roles.ClusterBackup, node)
				}
				continue L
			}
		}

		// Since we filter Pods that run gravity-site (i.e. master nodes) above,
		// here only the regular nodes are left
		roles.Regular = append(roles.Regular, node)
	}

	return roles, nil
}

// apiserverNode returns the node that runs the API server
func apiserverNode(ctx context.Context, nodes []Gravity) (api Gravity, other []Gravity, err error) {
	const apiserver = "leader.telekube.local"
	if len(nodes) < 1 {
		return nil, nil, trace.BadParameter("at least one node required")
	}

	apiserverAddr, err := ResolveInPlanet(ctx, nodes[0], apiserver)
	if err != nil {
		return nil, nil, trace.Wrap(err, "failed to resolve %v", apiserver)
	}

	other = make([]Gravity, 0, len(nodes)-1)
	for _, node := range nodes {
		if node.Node().PrivateAddr() == apiserverAddr {
			api = node
		} else {
			other = append(other, node)
		}
	}

	if api == nil {
		return nil, nil, trace.NotFound("no apiserver node found")
	}

	return api, other, nil
}
