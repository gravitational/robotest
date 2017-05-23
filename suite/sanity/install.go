package sanity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

// installReliability performs cyclic installs
// https://github.com/gravitational/gravity/issues/2251
func installReliability(ctx context.Context, t *testing.T, config *gravity.ProvisionerConfig, payload interface{}) {
	for _, size := range []uint{1, 3, 6} {
		gravity.Run(ctx, t, config, cycleInstall, cycleInstallParam{nodes: size, cycles: 15}, gravity.Parallel)
	}
}

type cycleInstallParam struct{ nodes, cycles uint }

func cycleInstall(ctx context.Context, t *testing.T, baseConfig *gravity.ProvisionerConfig, payload interface{}) {
	param, ok := payload.(cycleInstallParam)
	require.True(t, ok)

	cfg := baseConfig.WithNodes(param.nodes)
	nodes, destroyFn, err := gravity.Provision(ctx, t, cfg)
	require.NoError(t, err, "provision nodes")
	defer gravity.Destroy(t, destroyFn)

	var c uint
	for c = 1; c <= param.cycles; c++ {
		gravity.OfflineInstall(ctx, t, nodes)
		gravity.Uninstall(ctx, t, nodes)
	}
}
