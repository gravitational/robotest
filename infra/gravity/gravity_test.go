package gravity

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testStatusStr = []byte(`
Cluster:	nostalgicjones2725, created at Mon May 15 18:08 UTC (50 minutes ago)
    node-0 (10.40.2.4), Mon May 15 18:08 UTC
Application:		mattermost, version 2.2.0
Status:			expanding
Join token:		fac3b88014367fe4e98a8664755e2be4
Periodic updates:	Not Configured
Remote support:
    10.40.2.4: ON
Operation:
    operation_expand (d0264693-023d-40fb-b5e1-86aa7cdf9e35)
    started:	Mon May 15 18:40 UTC (19 minutes ago)
    initializing, 0% complete
`)

func TestGravityOutput(t *testing.T) {
	expectedStatus := &GravityStatus{
		Cluster:     "nostalgicjones2725",
		Application: "mattermost",
		Status:      "expanding",
		Token:       "fac3b88014367fe4e98a8664755e2be4",
		Nodes:       []string{"10.40.2.4"},
	}

	var status GravityStatus
	err := parseStatus(&status)(bufio.NewReader(bytes.NewReader(testStatusStr)))
	assert.NoError(t, err)
	assert.Equal(t, expectedStatus, &status, "parseStatus")
}
