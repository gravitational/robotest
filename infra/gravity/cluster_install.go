package gravity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/lib/utils"

	"github.com/gravitational/trace"
)

// OfflineInstall sets up cluster using nodes provided
func OfflineInstall(ctx context.Context, t *testing.T, nodes []Gravity, flavor, role string) error {
	if len(nodes) == 0 {
		return trace.Errorf("at least one node")
	}

	master := nodes[0]
	token := "ROBOTEST"

	errs := make(chan error, len(nodes))
	go func() {
		errs <- master.Install(ctx, InstallCmd{
			Token:  token,
			Flavor: flavor,
		})
	}()

	for _, node := range nodes[1:] {
		go func(n Gravity) {
			// TODO: how to properly define node role ?
			errs <- n.Join(ctx, JoinCmd{
				PeerAddr: master.Node().PrivateAddr(),
				Token:    token,
				Role:     role})
		}(node)
	}

	return trace.Wrap(utils.CollectErrors(ctx, errs))
}

// Uninstall makes nodes leave cluster and uninstall gravity
// it is not asserting internally
func Uninstall(ctx context.Context, t *testing.T, nodes []Gravity) error {
	errs := make(chan error, len(nodes))

	for _, node := range nodes {
		go func(n Gravity) {
			errs <- n.Uninstall(ctx)
		}(node)
	}

	return trace.Wrap(utils.CollectErrors(ctx, errs))
}
