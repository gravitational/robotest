package gravity

import (
	"testing"

	"github.com/gravitational/robotest/infra"
	log "github.com/sirupsen/logrus"
)

func TestGenerateOpsClusterDefn(t *testing.T) {
	cfg := ProvisionerConfig{
		Ops: &infra.OpsConfig{
			App:    "abc:1.2.3",
			Region: "us-east-2",
		},
		NodeCount: 5,
	}

	defn, err := generateClusterDefn(cfg, "test")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if defn != `
kind: cluster
version: v2
metadata:
  labels:
    Name: test
  name: test
spec:
  app: abc:1.2.3
  aws:
    keyName: ops
    region: us-east-2
  nodes:
  - profile: node
    count: 5
    instanceType: c4.2xlarge
  provider: aws` {
		t.Error("generated cluster configuration doesn't match expected output")
		log.Info(defn)
	}
}
