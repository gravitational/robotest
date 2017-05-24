package gravity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/lib/utils"

	"github.com/stretchr/testify/require"
)

const (
	defaultRole = "worker"
)

// OfflineInstall sets up cluster using nodes provided
func OfflineInstall(ctx context.Context, t *testing.T, nodes []Gravity) {
	require.NotZero(t, len(nodes), "at least 1 node")

	master := nodes[0]
	token := "ROBOTEST"

	errs := make(chan error, len(nodes))
	go func() {
		errs <- master.Install(ctx, InstallCmd{
			Token: token,
		})
	}()

	for _, node := range nodes[1:] {
		go func(n Gravity) {
			// TODO: how to properly define node role ?
			errs <- n.Join(ctx, JoinCmd{
				PeerAddr: master.Node().PrivateAddr(),
				Token:    token,
				Role:     defaultRole})
		}(node)
	}

	err := utils.CollectErrors(ctx, errs)
	require.NoError(t, err, "installation")

}

// Uninstall makes nodes leave cluster and uninstall gravity
func Uninstall(ctx context.Context, t *testing.T, nodes []Gravity) error {
	errs := make(chan error, len(nodes))

	for _, node := range nodes {
		go func(n Gravity) {
			errs <- n.Uninstall(ctx)
		}(node)
	}

	return utils.CollectErrors(ctx, errs)
}
