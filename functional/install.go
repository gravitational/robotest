package functional

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultRole = "worker"
)

func testOfflineInstall(ctx context.Context, t *testing.T, nodes []gravity.Gravity) {
	require.NotZero(t, len(nodes), "at least 1 node")

	master := nodes[0]
	token := "ROBOTEST"

	errs := make(chan error, len(nodes))
	go func() {
		errs <- master.Install(ctx, gravity.InstallCmd{
			DockerVolume: "/dev/sdd",
			Token:        token,
		})
	}()

	for _, node := range nodes[1:] {
		go func(n gravity.Gravity) {
			// TODO: how to properly define node role ?
			errs <- n.Join(ctx, master.Node().PrivateAddr(), token, defaultRole)
		}(node)
	}

	for range nodes {
		assert.NoError(t, <-errs)
	}

	// ensure all nodes see each other
	logFn := Logf(t, "offlineInstall")
	for _, node := range nodes {
		status, err := node.Status(ctx)
		require.NoError(t, err, "node status")
		logFn("node %s status=%+v", node.Node().Addr(), status)
		// TODO: how to properly verify?
	}
}
