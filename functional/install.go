package suite

import (
	"sync"
	"testing"

	. "github.com/gravitational/robotest/infra"

	"github.com/stretchr/testify/require"
)

func TestOfflineInstall(t *testing.T) {
	nodes, err := ProvisionNodes()
	require.NoError(t, err, "provision nodes")
}

func testOfflineInstall(t *testing.T, nodes []Node) {
	require.True(t, len(nodes) >= 2, "at least 2 nodes")

	master := nodes[0]
	token := makeUUID()
	err := master.Gravity().Install(InstallCmd{
		Token: token,
	})
	require.NoError(t, err, "gravity master installer")

	errs := make(chan error)
	for _, node := range nodes[1:] {
		go func() {
			errs <- node.Gravity().Join(master.PrivateAddr(), token, defaultRole)
		}()
	}

	for _, node := range nodes[1:] {
		err = <-errs
		require.NoError(t, err, "joining cluster")
	}

	// ensure all nodes see each other
	for _, node := range nodes {
		status, err := node.Gravity().Status()
		require.NoError(t, err, "node status")
		requireNodes(t, status, nodes)
	}
}
