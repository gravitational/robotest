package sanity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

type upgradeParam struct {
	// BaseInstallerURL is initial app installer URL
	BaseInstallerURL string
	// NodeCount is how many nodes
	NodeCount uint
	// Flavor is installer flavor corresponding to amount of nodes
	Flavor string
	// Role is standard node role as per app.yaml
	Role string
	// Timeouts are standard timeouts to use
	Timeouts gravity.OpTimeouts
}

func upgrade(param upgradeParam) gravity.TestFunc {
	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		cfg := baseConfig.WithNodes(param.NodeCount)

		nodes, destroyFn, err := gravity.Provision(ctx, t, cfg)
		require.NoError(t, err, "provision nodes")
		defer destroyFn(ctx, t)

		g := gravity.NewContext(ctx, t, param.Timeouts)
		g.OK("base installer", g.SetInstaller(nodes, param.BaseInstallerURL, "base"))
		g.OK("install", g.OfflineInstall(nodes, param.Flavor, param.Role))
		g.OK("status", g.Status(nodes))
		g.OK("upgrade installer", g.SetInstaller(nodes, cfg.InstallerURL, "upgrade"))
		g.OK("upgrade", g.Upgrade(nodes))
		g.OK("status", g.Status(nodes))
	}
}
