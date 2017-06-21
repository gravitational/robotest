package sanity

import (
	"flag"

	"github.com/gravitational/robotest/infra/gravity"
)

var defaultTimeouts = gravity.DefaultTimeouts

var baseInstallerURL = flag.String("sanity-base-installer", "s3://s3.gravitational.io/denis/telekube-3.53.10-installer.tar", "base application version")

var Basic = map[string]gravity.TestFunc{
	"provision1": install(installParam{
		NodeCount: 1, Flavor: "one", Timeouts: defaultTimeouts, Role: "node"}),
	"provision3": install(installParam{
		NodeCount: 3, Flavor: "three", Timeouts: defaultTimeouts, Role: "node"}),
	"provision6": install(installParam{
		NodeCount: 6, Flavor: "six", Timeouts: defaultTimeouts, Role: "node"}),

	"install1": install(installParam{
		NodeCount: 1, Flavor: "one", Timeouts: defaultTimeouts, Role: "node"}),
	"install3": install(installParam{
		NodeCount: 3, Flavor: "three", Timeouts: defaultTimeouts, Role: "node"}),
	"install6": install(installParam{
		NodeCount: 6, Flavor: "six", Timeouts: defaultTimeouts, Role: "node"}),

	"apiMasterLoss3": lossAndRecovery(lossAndRecoveryParam{
		InitialNodes: 3, InitialFlavor: "three", ReplaceNodeType: nodeApiMaster, Role: "node", Timeouts: defaultTimeouts}),
	"apiMasterLoss6": lossAndRecovery(lossAndRecoveryParam{
		InitialNodes: 6, InitialFlavor: "six", ReplaceNodeType: nodeApiMaster, Role: "node", Timeouts: defaultTimeouts}),

	"gsMasterLoss3": lossAndRecovery(lossAndRecoveryParam{
		InitialNodes: 3, InitialFlavor: "three", ReplaceNodeType: nodeGravitySiteMaster, Role: "node", Timeouts: defaultTimeouts}),
	"gsMasterLoss6": lossAndRecovery(lossAndRecoveryParam{
		InitialNodes: 6, InitialFlavor: "six", ReplaceNodeType: nodeGravitySiteMaster, Role: "node", Timeouts: defaultTimeouts}),

	"gsNodeLoss3": lossAndRecovery(lossAndRecoveryParam{
		InitialNodes: 3, InitialFlavor: "three", ReplaceNodeType: nodeGravitySiteNode, Role: "node", Timeouts: defaultTimeouts}),
	"gsNodeLoss6": lossAndRecovery(lossAndRecoveryParam{
		InitialNodes: 6, InitialFlavor: "six", ReplaceNodeType: nodeGravitySiteNode, Role: "node", Timeouts: defaultTimeouts}),

	"workerNodeLoss6": lossAndRecovery(lossAndRecoveryParam{
		InitialNodes: 6, InitialFlavor: "six", ReplaceNodeType: nodeRegularNode, Role: "node", Timeouts: defaultTimeouts}),

	"expand13": basicExpand(expandParam{
		InitialNodes: 1, TargetNodes: 3, InitialFlavor: "one", Role: "node", Timeouts: defaultTimeouts}),
	"expand23": basicExpand(expandParam{
		InitialNodes: 2, TargetNodes: 3, InitialFlavor: "two", Role: "node", Timeouts: defaultTimeouts}),
	"expand36": basicExpand(expandParam{
		InitialNodes: 3, TargetNodes: 6, InitialFlavor: "one", Role: "node", Timeouts: defaultTimeouts}),

	"upgrade1": upgrade(upgradeParam{
		NodeCount: 1, Flavor: "one", Role: "node", Timeouts: defaultTimeouts, BaseInstallerURL: *baseInstallerURL}),
	"upgrade3": upgrade(upgradeParam{
		NodeCount: 3, Flavor: "three", Role: "node", Timeouts: defaultTimeouts, BaseInstallerURL: *baseInstallerURL}),
	"upgrade6": upgrade(upgradeParam{
		NodeCount: 6, Flavor: "six", Role: "node", Timeouts: defaultTimeouts, BaseInstallerURL: *baseInstallerURL}),
}
