package sanity

import (
	"context"
	"fmt"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/stretchr/testify/require"
)

type expandParam struct {
	installParam
	// TargetNodes is how many nodes cluster should have after expand
	ToNodes uint `json:"to" validate:"required,gte=3"`
}

// basicExpand installs an initial cluster and then expands it to given number of nodes
func basicExpand(p interface{}) (gravity.TestFunc, error) {
	param := p.(expandParam)

	return func(ctx context.Context, t *testing.T, baseConfig gravity.ProvisionerConfig) {
		config := baseConfig.WithNodes(param.ToNodes)

		nodes, destroyFn, err := gravity.Provision(ctx, t, config)
		require.NoError(t, err, "provision nodes")
		defer destroyFn(ctx, t)

		g := gravity.NewContext(ctx, t, param.Timeouts)

		g.OK("download installer", g.SetInstaller(nodes, config.InstallerURL, "install"))
		g.OK(fmt.Sprintf("install on %d node", param.NodeCount),
			g.OfflineInstall(nodes[0:param.NodeCount], param.InstallParam))
		g.OK("status", g.Status(nodes[0:param.NodeCount]))
		g.OK("time sync", g.CheckTimeSync(nodes))

		g.OK(fmt.Sprintf("expand to %d nodes", param.ToNodes),
			g.Expand(nodes[0:param.NodeCount], nodes[param.NodeCount:param.ToNodes], param.Role))
		g.OK("status", g.Status(nodes[0:param.ToNodes]))
	}, nil
}
