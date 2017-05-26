package gravity

import (
	"context"

	"github.com/gravitational/robotest/lib/utils"

	"github.com/gravitational/trace"
)

// OfflineInstall sets up cluster using nodes provided
func (c TestContext) OfflineInstall(nodes []Gravity, flavor, role string) error {
	ctx, cancel := context.WithTimeout(c.parent, withDuration(c.timeouts.Install, len(nodes)))
	defer cancel()

	if len(nodes) == 0 {
		return trace.NotFound("at least one node")
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
			err := n.Join(ctx, JoinCmd{
				PeerAddr: master.Node().PrivateAddr(),
				Token:    token,
				Role:     role})
			if err != nil {
				n.Logf("Join failed, will cancel install")
				cancel()
			}
			errs <- err
		}(node)
	}

	return trace.Wrap(utils.CollectErrors(ctx, errs))
}

// Uninstall makes nodes leave cluster and uninstall gravity
// it is not asserting internally
func (c TestContext) Uninstall(nodes []Gravity) error {
	ctx, cancel := context.WithTimeout(c.parent, withDuration(c.timeouts.Install, len(nodes)))
	defer cancel()

	errs := make(chan error, len(nodes))

	for _, node := range nodes {
		go func(n Gravity) {
			errs <- n.Uninstall(ctx)
		}(node)
	}

	return trace.Wrap(utils.CollectErrors(ctx, errs))
}
