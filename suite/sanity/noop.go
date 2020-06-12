/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
