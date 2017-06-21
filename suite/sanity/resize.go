package sanity

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"time"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

const (
	// K8S API master node
	nodeApiMaster = "apimaster"
	// Gravity Site master node
	nodeGravitySiteMaster = "gsmaster"
	// One of the GravitySite nodes
	nodeGravitySiteNode = "gsnode"
	// Regular node
	nodeRegularNode = "regular"
)

type lossAndRecoveryParam struct {
	// Timeouts is per-node operation timeout value
	Timeouts gravity.OpTimeouts
	// Role is node role
	Role string
	// InitialFlavor is equivalent to InitialNodes node
	InitialFlavor string
	// InitialNodes how many nodes
	InitialNodes uint
	// ReplaceNodes is how many nodes to loose and recover
	ReplaceNodes uint
	// ReplaceNodeType : see killXXX constants
	ReplaceNodeType string
}

type expandParam struct {
	// Timeouts is per-node operation timeout value
	Timeouts gravity.OpTimeouts
	// Role is node role
	Role string
	// InitialFlavor is equivalent to 1 node
	InitialFlavor string
	// InitialNodes is how many nodes on first install
	InitialNodes uint
	// TargetNodes is how many nodes cluster should have after expand
	TargetNodes uint
}

const (
	// reasonable timeframe a node should report its status including all overheads
	statusTimeout = time.Second * 30
	// reasonable timeframe infrastructure should be provisioned
	provisioningTimeout = time.Minute * 20
	// minTimeout to guard against forgotten params or ridiculously small values
	minTimeout = time.Minute * 5
)

// basicExpand installs an initial cluster and then expands it to given number of nodes
func basicExpand(param expandParam) gravity.TestFunc {
	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		config := baseConfig.WithNodes(param.TargetNodes)

		nodes, destroyFn, err := gravity.Provision(ctx, t, config)
		require.NoError(t, err, "provision nodes")
		defer destroyFn(ctx, t)

		g := gravity.NewContext(ctx, t, param.Timeouts)

		g.OK("download installer", g.SetInstaller(nodes, config.InstallerURL, "install"))
		g.OK(fmt.Sprintf("install on %d node", param.InitialNodes),
			g.OfflineInstall(nodes[0:param.InitialNodes], param.InitialFlavor, param.Role))
		g.OK("status", g.Status(nodes[0:param.InitialNodes]))
		g.OK("time sync", g.CheckTimeSync(nodes))

		g.OK(fmt.Sprintf("expand to %d nodes", param.TargetNodes),
			g.Expand(nodes[0:param.InitialNodes], nodes[param.InitialNodes:param.TargetNodes], param.Role))
		g.OK("status", g.Status(nodes[0:param.TargetNodes]))
	}
}

// lossAndRecovery installs cluster then fails one of the nodes, and then removes it
// There are several modes how nodes could be removed:
// 1. forcible or not : i.e. whether node is shut down first or not
// 2. whether node is master, or one of gravity-site members, or regular node
// 3.
// 4.

func lossAndRecovery(param lossAndRecoveryParam) gravity.TestFunc {
	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		config := baseConfig.WithNodes(param.InitialNodes + 1)

		allNodes, destroyFn, err := gravity.Provision(ctx, t, config)
		require.NoError(t, err, "provision nodes")
		defer destroyFn(ctx, t)

		g := gravity.NewContext(ctx, t, param.Timeouts)

		g.OK("download installer", g.SetInstaller(allNodes, config.InstallerURL, "install"))

		installed := allNodes[0:param.InitialNodes]
		g.OK("install", g.OfflineInstall(installed, param.InitialFlavor, param.Role))

		g.Sleep("between install and node loss", time.Minute*2)

		before, err := g.NodesByRole(installed)
		g.OK("node roles", err)
		g.Logf("Cluster Roles: %+v", before)

		var remaining []gravity.Gravity
		switch param.ReplaceNodeType {
		case nodeApiMaster:
			remaining = excludeNode(installed, before.ApiMaster)
			err = g.NodeLoss(remaining, before.ApiMaster)
		case nodeGravitySiteNode:
			require.Len(t, before.GravitySiteBackup, 2)

			// avoid picking up ApiMaster, as it'll become a very different test then
			idx := 0
			if before.GravitySiteBackup[idx] == before.ApiMaster {
				idx = 1
			}
			remaining = excludeNode(installed, before.GravitySiteBackup[idx])
			err = g.NodeLoss(remaining, before.GravitySiteBackup[idx])
		case nodeRegularNode:
			require.NotEmpty(t, before.Regular)
			idx := rand.Intn(len(before.Regular))
			remaining = excludeNode(installed, before.Regular[idx])
			err = g.NodeLoss(remaining, before.Regular[idx])
		default:
			t.Fatalf("unexpected node role type %s", param.ReplaceNodeType)
		}
		g.OK("node loss", err)

		g.Sleep("between node loss and recovery", time.Minute)

		g.OK("replace node", g.Expand(remaining, allNodes[param.InitialNodes:param.InitialNodes+1], param.Role))

		after, err := g.NodesByRole(append(remaining, allNodes[param.InitialNodes]))
		g.OK("node role after replacement", err)
		g.Logf("Cluster Roles: %+v", after)

		require.NotNil(t, after.ApiMaster, "api master")
		require.Len(t, after.GravitySiteBackup, 2)
		require.Len(t, after.Regular, int(param.InitialNodes-3))
	}
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
