package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"
)

var defaultTimeouts = gravity.DefaultTimeouts

var Basic = map[string]gravity.TestFunc{
	"install_1": install(installParam{
		Flavors: map[uint]string{1: "one"}, Timeouts: defaultTimeouts, Role: "node"}),
	"install_3": install(installParam{
		Flavors: map[uint]string{3: "three"}, Timeouts: defaultTimeouts, Role: "node"}),
	"install_6": install(installParam{
		Flavors: map[uint]string{6: "six"}, Timeouts: defaultTimeouts, Role: "node"}),
	"install_136": install(installParam{
		Flavors:  map[uint]string{1: "one", 3: "three", 6: "six"},
		Timeouts: defaultTimeouts, Role: "node"}),
	"basicResize": basicResize(resizeParam{
		InitialFlavor: "one", Role: "node", Timeouts: defaultTimeouts}),
}
