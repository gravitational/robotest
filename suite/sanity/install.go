package sanity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

// installReliability performs cyclic installs
// https://github.com/gravitational/gravity/issues/2251
func installInCycles(cycles uint, sizes []uint) gravity.TestFunc {
	return func(ctx context.Context, t *testing.T, baseConfig *gravity.ProvisionerConfig) {
		cycleInstall(ctx, t, baseConfig, cycles, sizes)
	}
}

func cycleInstall(ctx context.Context, t *testing.T, baseConfig *gravity.ProvisionerConfig, cycles uint, sizes []uint) {
	install := func(cfg *gravity.ProvisionerConfig) func(*testing.T) {
		return func(t *testing.T) {
			t.Parallel()

			nodes, destroyFn, err := gravity.Provision(ctx, t, cfg)
			require.NoError(t, err, "provision nodes")
			defer gravity.Destroy(t, destroyFn)

			var c uint
			for c = 1; c <= cycles; c++ {
				gravity.Uninstall(ctx, t, nodes)
				gravity.OfflineInstall(ctx, t, nodes)
			}
		}
	}

	for _, nodes := range sizes {
		cfg := baseConfig.WithTag("cyclic").WithNodes(nodes)
		t.Run(cfg.Tag(), install(cfg))
	}
}
