package sanity

import (
	"github.com/gravitational/robotest/infra/gravity"
)

var Basic = map[string]gravity.TestFunc{
	"install_1":   installInCycles(2, []uint{1}),
	"install_10":  installInCycles(10, []uint{1, 3, 6}),
	"basicResize": basicResize,
}
