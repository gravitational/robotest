package suite

import (
	"flag"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var configFile = flag.String("config", "", "cloud config file in YAML")

func TestAll(t *testing.T) {
	config, err := parseConfig(*configFile)
	require.NoError(t, err, "config file")

	for _, os := range []string{"ubuntu", "centos"} {
		t.Run(fmt.Sprintf("OfflineInstallSuite os=%s", os),
			func(t *testing.T) {
				t.Parallel()
				testOfflineAll(t, os)
			})
	}
}

func testOfflineAll(t *testing.T) {
	vms, err := provision(config, 6, os)
	require.NoError(t, err, "provision nodes")
	defer teardown(vms)

	ok := t.Run("installOffline 1 node", func(t *testing.T) {
		testOfflineInstall(t, nodes[0:1])
	})
	require.True(t, ok, "installOffline 1 node")

	ok = t.Run("expandOffline to 6 nodes", func(t *testing.T) {
		testExpand(t, nodes[0:1], nodes[1:6])
	})
	require.True(t, ok, "expandOffline to 6 nodes")

	ok = t.Run("shrinkOffline to 3 nodes", func(t *testing.T) {
		testShrink(t, nodes[0:3], nodes[4:6])
	})
	require.True(t, ok, "shrinkOffline to 3 nodes")
}
