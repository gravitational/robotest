package sanity

import (
	"time"

	"github.com/gravitational/robotest/infra/gravity"
)

type noopParam struct {
	SleepSeconds int  `json:"sleep"`
	Fail         bool `json:"fail"`
}

func noop(p interface{}) (gravity.TestFunc, error) {
	param := p.(noopParam)

	return func(g *gravity.TestContext, baseConfig gravity.ProvisionerConfig) {
		select {
		case <-g.Context().Done():
			g.Logger().Infof("context cancel")
		case <-time.After(time.Second * time.Duration(param.SleepSeconds)):
			g.Logger().Info("timer elapsed")
		}
		if param.Fail {
			g.FailNow()
		}
	}, nil
}
