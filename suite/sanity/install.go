package sanity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

type installParam struct {
	gravity.InstallParam
	// NodeCount is how many nodes
	NodeCount uint `json:"nodes" validate:"gte=1"`
	// Timeout defines operation timeouts
	Timeouts gravity.OpTimeouts
}

func install(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		cfg := baseConfig.WithNodes(param.NodeCount)
		nodes, destroyFn, err := gravity.Provision(ctx, t, cfg)
		require.NoError(t, err, "provision nodes")
		defer destroyFn(ctx, t)

		g := gravity.NewContext(ctx, t, param.Timeouts)
		g.OK("download installer", g.SetInstaller(nodes, cfg.InstallerURL, "install"))
		g.OK("install", g.OfflineInstall(nodes, param.InstallParam))
		g.OK("status", g.Status(nodes))
	}, nil
}

func provision(p interface{}) (gravity.TestFunc, error) {
	param := p.(installParam)

	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		cfg := baseConfig.WithNodes(param.NodeCount)
		nodes, destroyFn, err := gravity.Provision(ctx, t, cfg)
		require.NoError(t, err, "provision nodes")
		defer destroyFn(ctx, t)

		g := gravity.NewContext(ctx, t, param.Timeouts)
		g.OK("download installer", g.SetInstaller(nodes, cfg.InstallerURL, "install"))
	}, nil
}
