package sanity

import (
	"time"

	"github.com/gravitational/robotest/infra/gravity"
)

var Basic = map[string]gravity.TestFunc{
	"install_1_1": installInCycles(cycleInstallParam{
		Cycles: 1, Flavors: map[uint]string{1: "one"}, Role: "node"}),
	"install_1_3": installInCycles(cycleInstallParam{
		Cycles: 1, Flavors: map[uint]string{3: "three"}, Role: "node"}),
	"install_1_6": installInCycles(cycleInstallParam{
		Cycles: 1, Flavors: map[uint]string{6: "six"}, Role: "node"}),
	"install_10_136": installInCycles(cycleInstallParam{
		Cycles: 10, Flavors: map[uint]string{1: "one", 3: "three", 6: "six"}, Role: "node"}),
	"basicResize": basicResize(resizeParam{
		InitialFlavor: "one", Role: "node", ReasonableTimeout: time.Minute * 15}),
}
