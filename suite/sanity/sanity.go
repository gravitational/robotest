package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"
)

var defaultTimeouts = gravity.DefaultTimeouts

var Basic = map[string]gravity.TestFunc{
	"install_1_1": installInCycles(cycleInstallParam{
		Cycles: 1, Flavors: map[uint]string{1: "one"}, Timeouts: defaultTimeouts, Role: "node"}),
	"install_10_1": installInCycles(cycleInstallParam{
		Cycles: 10, Flavors: map[uint]string{1: "one"}, Timeouts: defaultTimeouts, Role: "node"}),
	"install_1_3": installInCycles(cycleInstallParam{
		Cycles: 1, Flavors: map[uint]string{3: "three"}, Timeouts: defaultTimeouts, Role: "node"}),
	"install_10_3": installInCycles(cycleInstallParam{
		Cycles: 10, Flavors: map[uint]string{3: "three"}, Timeouts: defaultTimeouts, Role: "node"}),
	"install_1_6": installInCycles(cycleInstallParam{
		Cycles: 1, Flavors: map[uint]string{6: "six"}, Timeouts: defaultTimeouts, Role: "node"}),
	"install_1_136": installInCycles(cycleInstallParam{
		Cycles: 1, Flavors: map[uint]string{1: "one", 3: "three", 6: "six"},
		Timeouts: defaultTimeouts, Role: "node"}),
	"install_10_136": installInCycles(cycleInstallParam{
		Cycles: 10, Flavors: map[uint]string{1: "one", 3: "three", 6: "six"},
		Timeouts: defaultTimeouts, Role: "node"}),
	"basicResize": basicResize(resizeParam{
		InitialFlavor: "one", Role: "node", Timeouts: defaultTimeouts}),
}
