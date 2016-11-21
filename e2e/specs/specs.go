package specs

import (
	"github.com/gravitational/robotest/infra"
	"github.com/sclevine/agouti"
)

type pageFunc func() *agouti.Page
type clusterFunc func() infra.Infra
