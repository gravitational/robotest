package sanity

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/trace"

	"github.com/stretchr/testify/require"
)

const (
	// K8S API master node
	nodeApiMaster = "apimaster"
	// Gravity Site master node
	nodeClusterMaster = "clmaster"
	// One of the GravitySite nodes
	nodeClusterBackup = "clbackup"
	// Regular node
	nodeRegularNode = "worker"
)

type lossAndRecoveryParam struct {
	installParam
	// ReplaceNodes is how many nodes to loose and recover
	ReplaceNodes uint `json:"replace" validate:"gte=0"`
	// ReplaceNodeType : see killXXX constants
	ReplaceNodeType string `json:"kill" validate:"required,eq=apimaster|clmaster|clbackup|worker"`
	// ExpandBeforeShrink is whether to expand cluster before removing dead node
	ExpandBeforeShrink bool `json:"expand_before_shrink" validate:"required"`
	// PowerOff is whether to power off node before remove
	PowerOff bool `json:"pwroff_before_remove" validate:"required"`
}

func lossAndRecoveryVariety(p interface{}) (gravity.TestFunc, error) {
	template := p.(lossAndRecoveryParam)

	var exp map[bool]string = map[bool]string{true: "expBfr", false: "expAft"}
	var pwr map[bool]string = map[bool]string{true: "pwrOff", false: "pwrOn"}

	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		for _, nodeRoleType := range []string{nodeApiMaster, nodeClusterMaster, nodeClusterBackup, nodeRegularNode} {
			for _, powerOff := range []bool{true, false} {
				for _, expandBeforeShrink := range []bool{true, false} {
					cfg := baseConfig.WithTag(fmt.Sprintf("%s-%s-%s", nodeRoleType, exp[expandBeforeShrink], pwr[powerOff]))
					param := template
					param.ExpandBeforeShrink = expandBeforeShrink
					param.ReplaceNodeType = nodeRoleType
					param.PowerOff = powerOff
					fun, err := lossAndRecovery(param)
					if err != nil {
						t.Fatalf("got error for configuration %v: %v", param, err)
					}
					gravity.Run(ctx, t, cfg, fun, gravity.Parallel)
				}
			}
		}
	}, nil
}

// lossAndRecovery installs cluster then fails one of the nodes, and then removes it
func lossAndRecovery(p interface{}) (gravity.TestFunc, error) {
	param := p.(lossAndRecoveryParam)

	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		config := baseConfig.WithNodes(param.NodeCount + 1)

		allNodes, destroyFn, err := gravity.Provision(ctx, t, config)
		require.NoError(t, err, "provision nodes")
		defer destroyFn(ctx, t)

		g := gravity.NewContext(ctx, t, param.Timeouts)

		g.OK("download installer", g.SetInstaller(allNodes, config.InstallerURL, "install"))

		g.Logf("Loss and Recovery test param %+v", param)

		nodes := allNodes[0:param.NodeCount]
		g.OK("install", g.OfflineInstall(nodes, param.InstallParam))
		g.OK("install status", g.Status(nodes))

		nodes, removed, err := removeNode(g, t, nodes, param.ReplaceNodeType, param.PowerOff)
		g.OK(fmt.Sprintf("node for removal=%v, poweroff=%v", removed, param.PowerOff), err)

		now := time.Now()
		g.OK("wait for readiness", g.Status(nodes))
		g.Logf("It took %v for cluster to become available", time.Since(now))

		if param.ExpandBeforeShrink {
			g.OK("add node",
				g.Expand(nodes, allNodes[param.NodeCount:param.NodeCount+1], param.Role))
			nodes = append(nodes, allNodes[param.NodeCount])

			roles, err := g.NodesByRole(nodes)
			g.OK("node role after expand", err)
			g.Logf("Roles after expand: %+v", roles)

			g.OK("remove old node", g.RemoveNode(nodes, removed))
		} else {
			g.OK("remove lost node", g.RemoveNode(nodes, removed))

			roles, err := g.NodesByRole(nodes)
			g.OK("node role after remove", err)
			g.Logf("Roles after remove: %+v", roles)

			g.OK("replace node",
				g.Expand(nodes, allNodes[param.NodeCount:param.NodeCount+1], param.Role))
			nodes = append(nodes, allNodes[param.NodeCount])
		}

		roles, err := g.NodesByRole(nodes)
		g.OK("final node roles", err)
		g.Logf("Final Cluster Roles: %+v", roles)

		g.Logf("Cluster Roles: %+v", roles)
		require.NotNil(t, roles.ApiMaster, "api master")
		require.Len(t, roles.ClusterBackup, 2)
		require.Len(t, roles.Regular, int(param.NodeCount-3))
	}, nil
}

func removeNode(g gravity.TestContext, t *testing.T,
	nodes []gravity.Gravity,
	nodeRoleType string, powerOff bool) (remaining []gravity.Gravity, removed gravity.Gravity, err error) {

	roles, err := g.NodesByRole(nodes)
	g.OK("node roles", err)
	g.Logf("Cluster Roles: %+v", roles)

	switch nodeRoleType {
	case nodeApiMaster:
		removed = roles.ApiMaster
	case nodeClusterMaster:
		if roles.ApiMaster == roles.ClusterMaster {
			g.Logf("API and Cluster masters reside on same node %s, will try to split", roles.ApiMaster.String())
			g.OK("cluster master relocation", gravity.RelocateClusterMaster(g.Context(), roles.ApiMaster))
			return removeNode(g, t, nodes, nodeRoleType, powerOff)
		}
		g.Require("gravity-site master != apiserver", roles.ApiMaster != roles.ClusterMaster)
		removed = roles.ClusterMaster
	case nodeClusterBackup:
		g.Require("2 cluster backup nodes", len(roles.ClusterBackup) == 2)
		// avoid picking up ApiMaster, as it'll become a very different test then
		idx := 0
		if roles.ClusterBackup[idx] == roles.ApiMaster {
			idx = 1
		}
		removed = roles.ClusterBackup[idx]
	case nodeRegularNode:
		require.NotEmpty(t, roles.Regular)
		removed = roles.Regular[rand.Intn(len(roles.Regular))]
	default:
		t.Fatalf("unexpected node role type %s", nodeRoleType)
	}

	remaining = excludeNode(nodes, removed)

	if powerOff {
		ctx, cancel := context.WithTimeout(g.Context(), time.Minute)
		defer cancel()
		err = removed.PowerOff(ctx, gravity.Graceful(false))
	}

	return remaining, removed, trace.Wrap(err)
}

func excludeNode(nodes []gravity.Gravity, excl gravity.Gravity) []gravity.Gravity {
	out := []gravity.Gravity{}
	for _, node := range nodes {
		if excl != node {
			out = append(out, node)
		}
	}
	return out
}
