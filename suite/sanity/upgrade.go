package sanity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

type upgradeParam struct {
	installParam
	// BaseInstallerURL is initial app installer URL
	BaseInstallerURL string `json:"upgrade_from" validate:"required"`
}

func upgrade(p interface{}) (gravity.TestFunc, error) {
	param := p.(upgradeParam)

	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		cfg := baseConfig.WithNodes(param.NodeCount)

		nodes, destroyFn, err := gravity.Provision(ctx, t, cfg)
		require.NoError(t, err, "provision nodes")
		defer destroyFn(ctx, t)

		g := gravity.NewContext(ctx, t, param.Timeouts)
		g.OK("base installer", g.SetInstaller(nodes, param.BaseInstallerURL, "base"))
		g.OK("install", g.OfflineInstall(nodes, param.InstallParam))
		g.OK("status", g.Status(nodes))
		g.OK("upgrade", g.Upgrade(nodes, cfg.InstallerURL, "upgrade"))
		g.OK("status", g.Status(nodes))
	}, nil
}
