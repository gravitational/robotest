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
	"bufio"
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGravityOutput(t *testing.T) {
	var testStatusStr = []byte(`
{"cluster":{"application":{"repository":"gravitational.io","name":"telekube","version":"0.0.1"},"state":"active","domain":"testcluster","token":{"token":"fac3b88014367fe4e98a8664755e2be4","expires":"0001-01-01T00:00:00Z","type":"expand","account_id":"00000000-0000-0000-0000-000000000001","site_domain":"testcluster","operation_id":"","user_email":"agent@testcluster"},"operation":{"type":"operation_install","id":"55298dfd-2094-47a3-a787-8b2a546c0fd1","state":"completed","created":"2008-01-01T12:00:00.0Z","progress":{"message":"Operation has completed","completion":100,"created":"2008-01-01T12:05:00.0Z"}},"system_status":1,"nodes":[{"hostname":"node-0","advertise_ip":"10.40.2.4","role":"master","profile":"node","status":"healthy"},{"hostname":"node-2","advertise_ip":"10.40.2.5","role":"master","profile":"node","status":"healthy"},{"hostname":"node-1","advertise_ip":"10.40.2.7","role":"master","profile":"node","status":"healthy"},{"hostname":"node-5","advertise_ip":"10.40.2.6","role":"node","profile":"node","status":"healthy"},{"hostname":"node-3","advertise_ip":"10.40.2.3","role":"node","profile":"node","status":"healthy"},{"hostname":"node-4","advertise_ip":"10.40.2.2","role":"node","profile":"node","status":"healthy"}]}}
`)
	expectedStatus := &GravityStatus{
		Cluster: ClusterStatus{
			Cluster:      "testcluster",
			Application:  Application{Name: "telekube"},
			State:        "active",
			SystemStatus: 1,
			Token:        Token{Token: "fac3b88014367fe4e98a8664755e2be4"},
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

func TestHealthyStatusValidation(t *testing.T) {
	healthyStatus := GravityStatus{
		Cluster: ClusterStatus{
			Cluster:      "robotest",
			Application:  Application{Name: "telekube"},
			State:        "active",
			SystemStatus: 1,
			Token:        Token{Token: "ROBOTEST"},
			Nodes: []NodeStatus{
				NodeStatus{Addr: "10.1.2.3"},
				NodeStatus{Addr: "10.1.2.4"},
				NodeStatus{Addr: "10.1.2.5"},
			},
		},
	}
	err := checkActive(healthyStatus)
	assert.NoError(t, err)
}

// Test1523StatusValidation ensures expanding status is
// identified as "unsafe to proceed" by Robotest.
//
// Expands may be unexpectedly seen after install as discussed
// in https://github.com/gravitational/gravity/issues/1523.
func Test1523StatusValidation(t *testing.T) {
	nonActiveStatus := GravityStatus{
		Cluster: ClusterStatus{
			Cluster:     "robotest",
			Application: Application{Name: "telekube"},
			State:       "expanding",
			Token:       Token{Token: "ROBOTEST"},
			Nodes: []NodeStatus{
				NodeStatus{Addr: "10.1.2.3"},
			},
		},
	}
	err := checkActive(nonActiveStatus)
	assert.Error(t, err)
}

// Test1641StatusValidation ensures a particular status type seen
// in the field identified as degraded by Robotest.
//
// See https://github.com/gravitational/gravity/issues/1641 for more info.
func Test1641StatusValidation(t *testing.T) {
	f, err := os.Open("testdata/status-degraded-1641.json")
	assert.NoError(t, err)
	defer f.Close()

	var status GravityStatus
	err = parseStatus(&status)(bufio.NewReader(f))
	assert.NoError(t, err)

	err = checkNotDegraded(status)
	assert.Error(t, err)
}
