package gravity

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testStatusStr = []byte(`
{"cluster":{"application":{"repository":"gravitational.io","name":"telekube","version":"0.0.1"},"state":"active","domain":"testcluster","token":{"token":"fac3b88014367fe4e98a8664755e2be4","expires":"0001-01-01T00:00:00Z","type":"expand","account_id":"00000000-0000-0000-0000-000000000001","site_domain":"testcluster","operation_id":"","user_email":"agent@testcluster"},"operation":{"type":"operation_install","id":"55298dfd-2094-47a3-a787-8b2a546c0fd1","state":"completed","created":"2008-01-01T12:00:00.0Z","progress":{"message":"Operation has completed","completion":100,"created":"2008-01-01T12:05:00.0Z"}},"system_status":1,"nodes":[{"hostname":"node-0","advertise_ip":"10.40.2.4","role":"master","profile":"node","status":"healthy"},{"hostname":"node-2","advertise_ip":"10.40.2.5","role":"master","profile":"node","status":"healthy"},{"hostname":"node-1","advertise_ip":"10.40.2.7","role":"master","profile":"node","status":"healthy"},{"hostname":"node-5","advertise_ip":"10.40.2.6","role":"node","profile":"node","status":"healthy"},{"hostname":"node-3","advertise_ip":"10.40.2.3","role":"node","profile":"node","status":"healthy"},{"hostname":"node-4","advertise_ip":"10.40.2.2","role":"node","profile":"node","status":"healthy"}]}}
`)

func TestGravityOutput(t *testing.T) {
	expectedStatus := &GravityStatus{
		Cluster: ClusterStatus{
			Cluster:     "testcluster",
			Application: Application{Name: "telekube"},
			Status:      "active",
			Token:       Token{Token: "fac3b88014367fe4e98a8664755e2be4"},
			Nodes: []NodeStatus{
				NodeStatus{Addr: "10.40.2.4"},
				NodeStatus{Addr: "10.40.2.5"},
				NodeStatus{Addr: "10.40.2.7"},
				NodeStatus{Addr: "10.40.2.6"},
				NodeStatus{Addr: "10.40.2.3"},
				NodeStatus{Addr: "10.40.2.2"},
			},
		},
	}

	var status GravityStatus
	err := parseStatus(&status)(bufio.NewReader(bytes.NewReader(testStatusStr)))
	assert.NoError(t, err)
	assert.Equal(t, expectedStatus, &status, "parseStatus")
}
