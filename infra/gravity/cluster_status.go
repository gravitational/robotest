package gravity

import (
	"context"
	"sort"
	"time"

	sshutils "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"
	log "github.com/sirupsen/logrus"
)

// Status walks around all nodes and checks whether they all feel OK
func (c *TestContext) Status(nodes []Gravity) error {
	c.Logger().WithField("nodes", Nodes(nodes)).Info("Check status on nodes.")
	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.Status)
	defer cancel()

	retry := wait.Retryer{
		Attempts: 100,
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
		c.Logger().Warnf("Status not available on some nodes, will retry: %v.", err)
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

	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.Status)
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

	nodes, err := c.reorderNodesForCollection(ctx, nodes)
	if err != nil {
		c.Logger().WithError(err).Warn("Failed to reorder nodes for collection.")
		// nodes is still valid
	}

	c.Logger().WithField("nodes", nodes).Debug("Collecting logs from nodes.")
	firstNodeArgs := []string{"--filter=system", "--filter=kubernetes"}
	nodeArgs := []string{"--filter=system"}
	err = c.collectLogsFromNodes(ctx, nodes, prefix, firstNodeArgs, nodeArgs)
	return trace.Wrap(err)
}

func (c *TestContext) reorderNodesForCollection(ctx context.Context, nodes []Gravity) ([]Gravity, error) {
	api, other, err := apiserverNode(ctx, nodes)
	if err != nil {
		// Return nodes unaltered in case of error
		return nodes, trace.Wrap(err)
	}
	return append([]Gravity{api}, other...), nil
}

func (c *TestContext) collectLogsFromNodes(ctx context.Context, nodes []Gravity, prefix string, firstNodeArgs, nodeArgs []string) error {
	errors := make(chan error, len(nodes))
	go func(node Gravity) {
		localPath, err := node.CollectLogs(ctx, prefix, firstNodeArgs...)
		node.Logger().WithFields(log.Fields{
			log.ErrorKey: err,
			"path":       localPath,
		}).Error("Fetching node logs.")
		errors <- err
	}(nodes[0])
	for _, node := range nodes[1:] {
		node := node
		go func() {
			localPath, err := node.CollectLogs(ctx, prefix, nodeArgs...)
			node.Logger().WithFields(log.Fields{
				log.ErrorKey: err,
				"path":       localPath,
			}).Error("Fetching node logs.")
			errors <- err
		}()
	}
	return trace.Wrap(utils.CollectErrors(ctx, errors))
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
	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.Status)
	defer cancel()

	roles = &ClusterNodesByRole{}
	roles.ApiMaster, roles.Other, err = apiserverNode(ctx, nodes)
	if err != nil {
		return nil, trace.Wrap(err)
	}

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

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Node().PrivateAddr() == apiserverAddr
	})
	api = nodes[0]
	if api.Node().PrivateAddr() != apiserverAddr {
		return nil, nil, trace.NotFound("no apiserver node found")
	}

	return api, nodes[1:], nil
}
