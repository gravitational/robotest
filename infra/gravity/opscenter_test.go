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

package gravity

import (
	"testing"

	"github.com/gravitational/trace"
)

func TestParseClusterStatus(t *testing.T) {
	t.Run("NotFound", func(t *testing.T) {
		txt := `cluster kevin-ci.22.7 not found`

		status, err := parseClusterStatus("kevin-ci.22.7", []byte(txt))
		if !trace.IsNotFound(err) {
			t.Error("expected not found error:", err)
		}
		if status != "" {
			t.Error("expected empty status")
		}
	})

	t.Run("Status", func(t *testing.T) {
		txt := `kind: cluster
metadata:
  labels:
    Name: kevin-ci.22.2
  name: kevin-ci.22.2
spec:
  app: ci:1.0.0-ci.22
  aws:
    keyName: ops
    region: us-east-2
  nodes: null
  provider: aws
  status: failed
version: v2`

		status, err := parseClusterStatus("kevin-ci.22.2", []byte(txt))
		if err != nil {
			t.Error("unexpected error: ", err)
		}
		if status != "failed" {
			t.Error("unexpected status: ", status)
		}
	})
}
