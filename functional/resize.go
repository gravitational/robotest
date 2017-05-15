package suite

import (
	"testing"
)

func testExpand(t *testing.T, current, extra []Node) {
	joinAddr := current[0].PrivateAddr()
	status, err := current[0].Status()
	require.NoError(t, err, "cluster status")

	errs := make(chan error)
	for _, node := range extra {
		go func() {
			errs <- node.Gravity().Join(joinAddr, status.Token, defaultRole)
		}()
	}

	for _, node := range extra {
		require.NoError(t, <-errs, "cluster join")
	}

	all := append([]Node{}, current)
	all = append(all, extra)

	for _, node := range all {
		status, err := node.Gravity().Status()
		require.NoError(t, err, "node status")
		requireNodes(t, status, nodes)
	}
}
