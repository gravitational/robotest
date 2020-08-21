/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gravity

import (
	"bufio"
	"context"
	"encoding/json"

	"github.com/cenkalti/backoff"
	"golang.org/x/sync/errgroup"

	sshutils "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/robotest/lib/wait"
	"github.com/gravitational/trace"
)

// ClusterState* consts come from https://github.com/gravitational/gravity/blob/7.0.0/lib/ops/constants.go#L64-L93
const (
	// ClusterStateActive is healthy and not running any operations.
	ClusterStateActive = "active"
	// ClusterStateDegraded is unhealthy.
	ClusterStateDegraded = "degraded"
)

// SystemStatus comes from https://github.com/gravitational/satellite/blob/7.1.0/agent/proto/agentpb/agent.pb.go#L28-L32
type SystemStatus int

const (
	// SystemStatus_Unknown is rarely/never returned in practice but is defined here for parity with satellite.
	SystemStatus_Unknown SystemStatus = 0
	// SystemStatus_Running is satellite's healthy status.
	SystemStatus_Running SystemStatus = 1
	// SystemStatus_Degraded is satellite's unhealthy status.
	SystemStatus_Degraded SystemStatus = 2
)

var systemStatusNames = [...]string{"undefined", "running", "degraded"}

func (ss SystemStatus) String() string {
	return systemStatusNames[ss]
}

// UnmarshalJSON fulfills the Unmarshaler interface, allowing SystemStatus
// to perform custom JSON unmarshalling.
//
// Gravity 5.2.x+ marshals SystemStatus as an integer and Gravity 5.0.36-
// marshals it as a string. For more info, see:
//   https://github.com/gravitational/robotest/issues/247
//
// For Gravity's definitions and mappings of see:
//   https://github.com/gravitational/satellite/blob/5.0.2/agent/proto/agentpb/agent.proto#L36-L40
//   https://github.com/gravitational/satellite/blob/5.0.2/agent/proto/agentpb/agent.pb.go#L48-L63
//
// TODO: When Gravity 5.0.x support is no longer needed, this function can
// be removed and SystemStatus can be swapped to type int:
//
//   SystemStatus int `json:"system_status"`
func (s *SystemStatus) UnmarshalJSON(data []byte) error {
	var intStatus int
	err := json.Unmarshal(data, &intStatus)
	if err == nil {
		*s = SystemStatus(intStatus)
		return nil
	}
	// parsing as int failed, try parsing as a string and converting
	var strStatus string
	if err2 := json.Unmarshal(data, &strStatus); err2 != nil {
		return trace.BadParameter("unable to parse system_status %q as int or string: %v, %v", data, err, err2)
	}
	switch strStatus {
	case SystemStatus_Running.String():
		*s = SystemStatus_Running
	case SystemStatus_Degraded.String():
		*s = SystemStatus_Degraded
	case SystemStatus_Unknown.String():
		*s = SystemStatus_Unknown
	default:
		return trace.BadParameter("unknown system_status: %q", strStatus)
	}
	return nil
}

// parseStatus is a helper adapting JSON Unmarshaling to sshutils.OutputParseFn
func parseStatus(status *GravityStatus) sshutils.OutputParseFn {
	return func(r *bufio.Reader) error {
		decoder := json.NewDecoder(r)
		return trace.Wrap(decoder.Decode(status))
	}
}

// GravityStatus describes the status of the Gravity cluster
type GravityStatus struct {
	// Cluster describes the cluster status
	Cluster ClusterStatus `json:"cluster"`
}

// ClusterStatus describes the status of a Gravity cluster
type ClusterStatus struct {
	// Application defines the cluster application
	Application Application `json:"application"`
	// Cluster is the name of the cluster
	Cluster string `json:"domain"`
	// State is the cluster state
	State string `json:"state"`
	// SystemStatus is the cluster status
	SystemStatus SystemStatus `json:"system_status"`
	// Token is secure token which prevents rogue nodes from joining the cluster during installation
	Token Token `json:"token"`
	// Nodes describes the nodes in the cluster
	Nodes []NodeStatus `json:"nodes"`
}

// Application defines the cluster application
type Application struct {
	// Name is the name of the cluster application
	Name string `json:"name"`
}

// NodeStatus describes the status of a cluster node
type NodeStatus struct {
	// Addr is the advertised address of this cluster node
	Addr string `json:"advertise_ip"`
}

// Token describes the cluster join token
type Token struct {
	// Token is the join token value
	Token string `json:"token"`
}

// statusValidator returns nil if the Gravity Status is the expected status or an error otherwise.
type statusValidator func(s GravityStatus) error

// checkNotDegraded returns an error if the cluster status is Degraded.
//
// This function is a reimplementation of the logic in https://github.com/gravitational/gravity/blob/7.0.0/lib/status/status.go#L180-L185
func checkNotDegraded(s GravityStatus) error {
	if s.Cluster.State == ClusterStateDegraded {
		return trace.CompareFailed("cluster state %q", s.Cluster.State)
	}
	if s.Cluster.SystemStatus != SystemStatus_Running {
		return trace.CompareFailed("expected system_status %v, found %v", SystemStatus_Running, s.Cluster.SystemStatus)
	}
	return nil
}

// checkActive returns an error if the cluster is degraded or state != active.
func checkActive(s GravityStatus) error {
	if err := checkNotDegraded(s); err != nil {
		return trace.Wrap(err)
	}
	if s.Cluster.State != ClusterStateActive {
		return trace.CompareFailed("expected state %q, found %q", ClusterStateActive, s.Cluster.State)
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

// Status queries `gravity status` once from each node in nodes.
func (c *TestContext) Status(nodes []Gravity) (statuses []GravityStatus, err error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.NodeStatus)
	defer cancel()

	valueC := make(chan GravityStatus, len(nodes))
	g, ctx := errgroup.WithContext(ctx)
	for _, node := range nodes {
		node := node
		g.Go(func() error {
			status, err := node.Status(ctx)
			if err != nil {
				return trace.Wrap(err)
			}
			if status != nil {
				valueC <- *status
			}
			return nil
		})
	}
	err = g.Wait()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	close(valueC)
	for status := range valueC {
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
