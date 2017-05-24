package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"
)

var Basic = map[string]gravity.TestFunc{
	"install_1_1":    installInCycles(1, []uint{1}),
	"install_1_3":    installInCycles(1, []uint{3}),
	"install_1_6":    installInCycles(1, []uint{6}),
	"install_10_136": installInCycles(10, []uint{1, 3, 6}),
	"basicResize":    basicResize,
}
