package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"
)

var defaultTimeouts = gravity.DefaultTimeouts

const (
	baseInstallerURL = "s3://s3.gravitational.io/denis/fda9ea3-telekube-3.52.3-installer.tar.gz"
)

var Basic = map[string]gravity.TestFunc{
	"install1": install(installParam{
		Flavors: map[uint]string{1: "one"}, Timeouts: defaultTimeouts, Role: "node"}),
	"install3": install(installParam{
		Flavors: map[uint]string{3: "three"}, Timeouts: defaultTimeouts, Role: "node"}),
	"install6": install(installParam{
		Flavors: map[uint]string{6: "six"}, Timeouts: defaultTimeouts, Role: "node"}),
	"install136": install(installParam{
		Flavors:  map[uint]string{1: "one", 3: "three", 6: "six"},
		Timeouts: defaultTimeouts, Role: "node"}),

	"basicResize": basicResize(resizeParam{
		InitialFlavor: "one", Role: "node", Timeouts: defaultTimeouts}),

	"expand13": basicExpand(expandParam{
		InitialNodes: 1, TargetNodes: 3, InitialFlavor: "one", Role: "node", Timeouts: defaultTimeouts}),
	"expand23": basicExpand(expandParam{
		InitialNodes: 2, TargetNodes: 3, InitialFlavor: "two", Role: "node", Timeouts: defaultTimeouts}),
	"expand36": basicExpand(expandParam{
		InitialNodes: 3, TargetNodes: 6, InitialFlavor: "one", Role: "node", Timeouts: defaultTimeouts}),

	"upgrade1": upgrade(upgradeParam{
		NodeCount: 1, Flavor: "one", Role: "node", Timeouts: defaultTimeouts, BaseInstallerURL: baseInstallerURL}),
	"upgrade3": upgrade(upgradeParam{
		NodeCount: 3, Flavor: "three", Role: "node", Timeouts: defaultTimeouts, BaseInstallerURL: baseInstallerURL}),
	"upgrade6": upgrade(upgradeParam{
		NodeCount: 6, Flavor: "six", Role: "node", Timeouts: defaultTimeouts, BaseInstallerURL: baseInstallerURL}),
}
