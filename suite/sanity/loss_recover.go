package sanity

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
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
	// ReplaceNodeType : see killXXX constants
	ReplaceNodeType string `json:"kill" validate:"required,eq=apimaster|clmaster|clbackup|worker"`
	// ExpandBeforeShrink is whether to expand cluster before removing dead node
	ExpandBeforeShrink bool `json:"expand_before_shrink" validate:"required"`
	// PowerOff is whether to power off node before remove
	PowerOff bool `json:"pwroff_before_remove" validate:"required"`
}

func lossAndRecoveryVariety(p interface{}) (gravity.TestFunc, error) {
	template := lossAndRecoveryParam{installParam: p.(installParam)}

	var exp map[bool]string = map[bool]string{true: "expBfr", false: "expAft"}
	var pwr map[bool]string = map[bool]string{true: "pwrOff", false: "pwrOn"}

	nodeRoleTypes := []string{nodeApiMaster, nodeClusterMaster, nodeClusterBackup}
	if template.NodeCount > 3 {
		nodeRoleTypes = append(nodeRoleTypes, nodeRegularNode)
	}
	return func(g *gravity.TestContext, baseConfig gravity.ProvisionerConfig) {
		for _, nodeRoleType := range nodeRoleTypes {

			for _, powerOff := range []bool{true, false} {
				for _, expandBeforeShrink := range []bool{true, false} {
					cfg := baseConfig.WithTag(fmt.Sprintf("%s-%s-%s", nodeRoleType, exp[expandBeforeShrink], pwr[powerOff]))
					param := template
					param.ExpandBeforeShrink = expandBeforeShrink
					param.ReplaceNodeType = nodeRoleType
					param.PowerOff = powerOff
					fun, err := lossAndRecovery(param)
					if err != nil {
						g.Logger().WithFields(logrus.Fields{
							"param": param, "error": err,
						}).Error("configuration error")
						g.FailNow()
					}
					g.Run(fun, cfg, logrus.Fields{"param": param})
				}
			}
		}
	}, nil
}

// lossAndRecovery installs cluster then fails one of the nodes, and then removes it
func lossAndRecovery(p interface{}) (gravity.TestFunc, error) {
	param := p.(lossAndRecoveryParam)

	return func(g *gravity.TestContext, baseConfig gravity.ProvisionerConfig) {
		config := baseConfig.WithNodes(param.NodeCount + 1)

		cluster, err := g.Provision(config)
		g.OK("provision nodes", err)
		defer func() {
			g.Maybe("uninstall application", g.UninstallApp(cluster.Nodes))
			g.Maybe("destroy", cluster.Destroy())
		}()

		g.OK("download installer", g.SetInstaller(cluster.Nodes, config.InstallerURL, "install"))

		nodes := cluster.Nodes[0:param.NodeCount]
		g.OK("install", g.OfflineInstall(nodes, param.InstallParam))
		g.OK("install status", g.Status(nodes))

		nodes, removed, err := removeNode(g, nodes, param.ReplaceNodeType, param.PowerOff)
		g.OK(fmt.Sprintf("node for removal=%v, poweroff=%v", removed, param.PowerOff), err)

		now := time.Now()
		g.OK("wait for cluster to be ready", g.Status(nodes))
		g.Logger().WithFields(logrus.Fields{"nodes": nodes, "elapsed": fmt.Sprintf("%v", time.Since(now))}).
			Info("cluster is available")

		if param.ExpandBeforeShrink {
			g.OK("expand before shrinking",
				g.Expand(nodes, cluster.Nodes[param.NodeCount:param.NodeCount+1], param.InstallParam))
			nodes = append(nodes, cluster.Nodes[param.NodeCount])

			roles, err := g.NodesByRole(nodes)
			g.OK("node roles after expand", err)
			g.Logger().WithFields(logrus.Fields{"roles": roles, "nodes": nodes}).
				Info("roles after expand")

			g.OK("remove old node", g.RemoveNode(nodes, removed))
		} else {
			g.OK("remove lost node", g.RemoveNode(nodes, removed))

			roles, err := g.NodesByRole(nodes)
			g.OK("node role after remove", err)
			g.Logger().WithFields(logrus.Fields{"roles": roles, "nodes": nodes}).
				Info("Roles after remove")

			g.OK("replace node",
				g.Expand(nodes, cluster.Nodes[param.NodeCount:param.NodeCount+1], param.InstallParam))
			nodes = append(nodes, cluster.Nodes[param.NodeCount])
		}

		roles, err := g.NodesByRole(nodes)
		g.OK("final node roles", err)
		g.Logger().WithFields(logrus.Fields{"roles": roles, "nodes": nodes}).Info("Final Cluster Roles")

	}, nil
}

func removeNode(g *gravity.TestContext,
	nodes []gravity.Gravity,
	nodeRoleType string, powerOff bool) (remaining []gravity.Gravity, removed gravity.Gravity, err error) {

	roles, err := g.NodesByRole(nodes)
	g.OK("node roles", err)
	g.Logger().WithFields(logrus.Fields{"roles": roles, "nodes": nodes}).Info("Cluster Roles")

	switch nodeRoleType {
	case nodeApiMaster:
		removed = roles.ApiMaster
	case nodeClusterMaster:
		if roles.ApiMaster == roles.ClusterMaster {
			g.Logger().Warn("API and Cluster masters reside on same node, will try relocate")
			g.OK("cluster master relocation", gravity.RelocateClusterMaster(g.Context(), roles.ApiMaster))
			return removeNode(g, nodes, nodeRoleType, powerOff)
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
		g.Require("worker nodes exist", len(roles.Regular) > 0)
		removed = roles.Regular[rand.Intn(len(roles.Regular))]
	default:
		g.Logger().WithField("role", nodeRoleType).Error("unexpected node role")
		g.FailNow()
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
