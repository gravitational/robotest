package sanity

import (
	"time"

	"github.com/gravitational/robotest/infra/gravity"
)

func noop(p interface{}) (gravity.TestFunc, error) {
	return func(g gravity.TestContext, baseConfig gravity.ProvisionerConfig) {
		defer g.Logger().Info("deferred")
		select {
		case <-g.Context().Done():
			g.Logger().Infof("context cancel")
			g.Require("CANCEL", false)
		case <-time.After(time.Second * 6):
			g.Logger().Info("timer elapsed")
			g.Require("TIMER", false)
		}
	}, nil
}
