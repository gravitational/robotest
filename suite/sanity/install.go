package sanity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

type installParam struct {
	// NodeCount is how many nodes
	NodeCount uint
	// Flavor is installer flavor corresponding to amount of nodes
	Flavor string
	// Role is node role as defined in app.yaml
	Role string
	// Timeout defines operation timeouts
	Timeouts gravity.OpTimeouts
}

func install(param installParam) gravity.TestFunc {
	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		t.Parallel()

		cfg := baseConfig.WithNodes(param.NodeCount)
		nodes, destroyFn, err := gravity.Provision(ctx, t, cfg)
		require.NoError(t, err, "provision nodes")
		defer destroyFn(ctx, t)

		g := gravity.NewContext(ctx, t, param.Timeouts)
		g.OK("download installer", g.SetInstaller(nodes, cfg.InstallerURL, "install"))
		g.OK("install", g.OfflineInstall(nodes, param.Flavor, param.Role))
		g.OK("status", g.Status(nodes))
	}
}

func provision(param installParam) gravity.TestFunc {
	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		t.Parallel()

		cfg := baseConfig.WithNodes(param.NodeCount)
		nodes, destroyFn, err := gravity.Provision(ctx, t, cfg)
		require.NoError(t, err, "provision nodes")
		defer destroyFn(ctx, t)

		g := gravity.NewContext(ctx, t, param.Timeouts)
		g.OK("download installer", g.SetInstaller(nodes, cfg.InstallerURL, "install"))
	}
}
