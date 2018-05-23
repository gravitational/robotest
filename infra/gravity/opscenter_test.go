package gravity

import (
	"testing"

	"github.com/gravitational/trace"
)

func TestParseClusterStatus(t *testing.T) {
	t.Run("NotFound", func(t *testing.T) {
		txt := `cluster kevin-ci.22.7 not found`

		status, err := parseClusterStatus([]byte(txt))
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

		status, err := parseClusterStatus([]byte(txt))
		if err != nil {
			t.Error("unexpected error: ", err)
		}
		if status != "failed" {
			t.Error("unexpected status: ", status)
		}
	})
}
