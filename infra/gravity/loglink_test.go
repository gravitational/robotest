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
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRenderLogURL checks that robotest generates valid links to "new" style GCP logs (as of 2020-09)
func TestRenderLogURL(t *testing.T) {
	expected := "https://console.cloud.google.com/logs/query;query=severity%3E%3DINFO%0Alabels.__uuid__%3D%22504d3d56-1abe-43cb-a802-3ebc96367d47%22%0Alabels.__suite__%3D%22d165f1b1-f40e-4e5f-8014-7bbb713d5357%22;timeRange=2020-09-22T22:33:00Z%2F2020-09-22T23:33:00Z?project=kubeadm-167321"
	date, err := time.Parse(time.RFC3339, "2020-09-22T22:33:00.000Z")
	assert.NoError(t, err)
	project := "kubeadm-167321"
	uuid := "504d3d56-1abe-43cb-a802-3ebc96367d47"
	suite := "d165f1b1-f40e-4e5f-8014-7bbb713d5357"
	found := encodeLogLink(uuid, suite, project, date)
	assert.Equal(t, expected, found)
}
