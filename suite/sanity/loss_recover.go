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
	nodeApiMaster = "apim"
	// Gravity Site master node
	nodeGravitySiteMaster = "gsm"
	// One of the GravitySite nodes
	nodeGravitySiteNode = "gsn"
	// Regular node
	nodeRegularNode = "wrk"
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
	// ExpandBeforeShrink is whether to expand cluster before removing dead node
	ExpandBeforeShrink bool
	// PowerOff is whether to power off node before remove
	PowerOff bool
}

func lossAndRecoveryVariety(template lossAndRecoveryParam) gravity.TestFunc {
	var exp map[bool]string = map[bool]string{true: "expBfr", false: "expAft"}
	var pwr map[bool]string = map[bool]string{true: "pwrOff", false: "pwrOn"}

	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		for _, nodeRoleType := range []string{nodeGravitySiteMaster} { // , nodeApiMaster, nodeGravitySiteNode, nodeRegularNode} {
			for _, powerOff := range []bool{true, false} {
				for _, expandBeforeShrink := range []bool{true, false} {
					cfg := baseConfig.WithTag(fmt.Sprintf("%s-%s-%s", nodeRoleType, exp[expandBeforeShrink], pwr[powerOff]))
					param := template
					param.ExpandBeforeShrink = expandBeforeShrink
					param.ReplaceNodeType = nodeRoleType
					param.PowerOff = powerOff
					gravity.Run(ctx, t, cfg, lossAndRecovery(param), gravity.Parallel)
				}
			}
		}
	}
}

// lossAndRecovery installs cluster then fails one of the nodes, and then removes it
func lossAndRecovery(param lossAndRecoveryParam) gravity.TestFunc {
	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		config := baseConfig.WithNodes(param.InitialNodes + 1)

		allNodes, destroyFn, err := gravity.Provision(ctx, t, config)
		require.NoError(t, err, "provision nodes")
		defer destroyFn(ctx, t)

		g := gravity.NewContext(ctx, t, param.Timeouts)

		g.OK("download installer", g.SetInstaller(allNodes, config.InstallerURL, "install"))

		g.Logf("Loss and Recovery test param %+v", param)

		nodes := allNodes[0:param.InitialNodes]
		g.OK("install", g.OfflineInstall(nodes, param.InitialFlavor, param.Role))
		g.OK("install status", g.Status(nodes))

		nodes, removed, err := removeNode(g, t, nodes, param.ReplaceNodeType, param.PowerOff)
		g.OK(fmt.Sprintf("node for removal=%v, poweroff=%v", removed, param.PowerOff), err)

		now := time.Now()
		g.OK("wait for readiness", g.Status(nodes))
		g.Logf("It took %v for cluster to become available", time.Since(now))

		if param.ExpandBeforeShrink {
			g.OK("replace node",
				g.Expand(nodes, allNodes[param.InitialNodes:param.InitialNodes+1], param.Role))
			nodes = append(nodes, allNodes[param.InitialNodes])

			roles, err := g.NodesByRole(nodes)
			g.OK("node role after expand", err)
			g.Logf("Roles after expand: %+v", roles)

			g.OK("remove lost node", g.RemoveNode(nodes, removed))
		} else {
			g.OK("remove lost node", g.RemoveNode(nodes, removed))

			roles, err := g.NodesByRole(nodes)
			g.OK("node role after remove", err)
			g.Logf("Roles after remove: %+v", roles)

			g.OK("replace node",
				g.Expand(nodes, allNodes[param.InitialNodes:param.InitialNodes+1], param.Role))
			nodes = append(nodes, allNodes[param.InitialNodes])
		}

		roles, err := g.NodesByRole(nodes)
		g.OK("final node roles", err)
		g.Logf("Final Cluster Roles: %+v", roles)

		g.Logf("Cluster Roles: %+v", roles)
		require.NotNil(t, roles.ApiMaster, "api master")
		require.Len(t, roles.GravitySiteBackup, 2)
		require.Len(t, roles.Regular, int(param.InitialNodes-3))
	}
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
	case nodeGravitySiteMaster:
		g.Require("gravity-site master != apiserver", roles.ApiMaster != roles.GravitySiteMaster)
		removed = roles.GravitySiteMaster
	case nodeGravitySiteNode:
		require.Len(t, roles.GravitySiteBackup, 2)
		// avoid picking up ApiMaster, as it'll become a very different test then
		idx := 0
		if roles.GravitySiteBackup[idx] == roles.ApiMaster {
			idx = 1
		}
		removed = roles.GravitySiteBackup[idx]
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
		err = removed.PowerOff(ctx, gravity.Force)
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
