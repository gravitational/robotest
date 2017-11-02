package sanity

import (
	"fmt"
	"time"

	"github.com/gravitational/robotest/infra/gravity"

	"cloud.google.com/go/bigquery"
	"github.com/sirupsen/logrus"
)

type noopParam struct {
	SleepSeconds int  `json:"sleep"`
	Fail         bool `json:"fail"`
}

func (p noopParam) Save() (row map[string]bigquery.Value, insertID string, err error) {
	row = make(map[string]bigquery.Value)
	row["extra"] = fmt.Sprintf("sleep=%v fail=%v", p.SleepSeconds, p.Fail)

	return row, "", nil
}

func noopVariety(p interface{}) (gravity.TestFunc, error) {
	param := p.(noopParam)

	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		for i := 1; i < 10; i++ {
			p := param
			if i == 5 {
				p.Fail = true
			}
			fun, _ := noop(p)
			g.Run(fun, cfg.WithTag(fmt.Sprintf("%d%v", i, p.Fail)), logrus.Fields{"fail": p.Fail})
		}
	}, nil
}

func noop(p interface{}) (gravity.TestFunc, error) {
	param := p.(noopParam)

	return func(g *gravity.TestContext, baseConfig gravity.ProvisionerConfig) {
		select {
		case <-g.Context().Done():
			g.Logger().Errorf("context cancel")
			g.FailNow()
		case <-time.After(time.Second * time.Duration(param.SleepSeconds)):
			g.Logger().Info("timer elapsed")
		}
		if param.Fail {
			g.FailNow()
		}
	}, nil
}
