package sanity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

type cycleInstallParam struct {
	// Cycles is how many times to repeat install / uninstall cycle
	Cycles uint
	// Flavors defines mapping between number of nodes allocated and build flavor as defined in app.yaml
	Flavors map[uint]string
	// Role is node role as defined in app.yaml
	Role string
}

// installReliability performs cyclic installs
// https://github.com/gravitational/gravity/issues/2251
func installInCycles(param cycleInstallParam) gravity.TestFunc {
	return func(ctx context.Context, t *testing.T, baseConfig *gravity.ProvisionerConfig) {
		cycleInstall(ctx, t, baseConfig, param)
	}
}

func cycleInstall(ctx context.Context, t *testing.T, baseConfig *gravity.ProvisionerConfig, param cycleInstallParam) {
	install := func(cfg *gravity.ProvisionerConfig, flavor string) func(*testing.T) {
		return func(t *testing.T) {
			t.Parallel()

			nodes, destroyFn, err := gravity.Provision(ctx, t, cfg)
			require.NoError(t, err, "provision nodes")
			defer destroyFn(ctx, t)

			var c uint
			for c = 1; c <= param.Cycles; c++ {
				err := gravity.OfflineInstall(ctx, t, nodes, flavor, param.Role)
				require.NoError(t, err, "install cycle %d of %d", c, param.Cycles)

				err = gravity.Uninstall(ctx, t, nodes)
				require.NoError(t, err, "uninstall cycle %d of %d", c, param.Cycles)
			}
		}
	}

	for nodes, flavor := range param.Flavors {
		cfg := baseConfig.WithTag("cyclic").WithNodes(nodes)
		t.Run(cfg.Tag(), install(cfg, flavor))
	}
}
