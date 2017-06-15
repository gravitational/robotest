package sanity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

type installParam struct {
	// Flavors defines mapping between number of nodes allocated and build flavor as defined in app.yaml
	Flavors map[uint]string
	// Role is node role as defined in app.yaml
	Role string
	// Timeout defines operation timeouts
	Timeouts gravity.OpTimeouts
}

// installReliability performs cyclic installs
// https://github.com/gravitational/gravity/issues/2251
func install(param installParam) gravity.TestFunc {
	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		cycleInstall(ctx, t, baseConfig, param)
	}
}

func cycleInstall(baseContext context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig, param installParam) {
	installFn := func(cfg gravity.ProvisionerConfig, flavor string) func(*testing.T) {
		return func(t *testing.T) {
			t.Parallel()

			nodes, destroyFn, err := gravity.Provision(baseContext, t, cfg)
			require.NoError(t, err, "provision nodes")
			defer destroyFn(baseContext, t)

			g := gravity.NewContext(baseContext, t, param.Timeouts)
			g.OK("download installer", g.SetInstaller(nodes, cfg.InstallerURL, "install"))
			g.OK("install", g.OfflineInstall(nodes, flavor, param.Role))
			g.OK("status", g.Status(nodes))
			g.OK("uninstall", g.Uninstall(nodes))
		}
	}

	for nodes, flavor := range param.Flavors {
		cfg := baseConfig.WithNodes(nodes)
		t.Run(cfg.Tag(), installFn(cfg, flavor))
	}
}
