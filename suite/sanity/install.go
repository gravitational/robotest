package sanity

import (
	"context"
	"fmt"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

type cycleInstallParam struct {
	// Cycles is how many times to repeat install / uninstall cycle
	Cycles int
	// Flavors defines mapping between number of nodes allocated and build flavor as defined in app.yaml
	Flavors map[uint]string
	// Role is node role as defined in app.yaml
	Role string
	// Timeout defines operation timeouts
	Timeouts gravity.OpTimeouts
}

// installReliability performs cyclic installs
// https://github.com/gravitational/gravity/issues/2251
func installInCycles(param cycleInstallParam) gravity.TestFunc {
	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		cycleInstall(ctx, t, baseConfig, param)
	}
}

func cycleInstall(baseContext context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig, param cycleInstallParam) {
	installCycle := func(cfg gravity.ProvisionerConfig, flavor string) func(*testing.T) {
		return func(t *testing.T) {
			t.Parallel()

			nodes, destroyFn, err := gravity.Provision(baseContext, t, cfg)
			require.NoError(t, err, "provision nodes")
			defer destroyFn(baseContext, t)

			g := gravity.NewContext(baseContext, t, param.Timeouts)
			require.NoError(t, g.OfflineInstall(nodes, flavor, param.Role))
			require.NoError(t, g.Status(nodes))
			require.NoError(t, g.Uninstall(nodes))
		}
	}

	install := func(cfg gravity.ProvisionerConfig, flavor string) func(*testing.T) {
		return func(t *testing.T) {
			t.Parallel()
			for c := 1; c <= param.Cycles; c++ {
				lc := cfg.WithTag(fmt.Sprintf("c%d", c))
				t.Run(lc.Tag(), installCycle(lc, flavor))
			}
		}
	}

	for nodes, flavor := range param.Flavors {
		cfg := baseConfig.WithNodes(nodes)
		t.Run(cfg.Tag(), install(cfg, flavor))
	}
}
