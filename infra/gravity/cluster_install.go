package gravity

import (
	"context"

	"github.com/gravitational/robotest/lib/utils"

	"github.com/gravitational/trace"
)

// ProvisionInstaller deploys a specific installer
func (c TestContext) SetInstaller(nodes []Gravity, installerUrl string, tag string) error {
	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Install)
	defer cancel()

	errs := make(chan error, len(nodes))
	for _, node := range nodes {
		go func(node Gravity) {
			errs <- node.SetInstaller(ctx, installerUrl, tag)
		}(node)
	}

	_, err := utils.Collect(ctx, cancel, errs, nil)
	return trace.Wrap(err)
}

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
				n.Logf("Join failed: %v", err)
			}
			errs <- err
		}(node)
	}

	_, err := utils.Collect(ctx, cancel, errs, nil)
	c.t.Log("install failed: %v", err)
	return trace.Wrap(err)
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

// Upgrade tries to perform an upgrade procedure on all nodes
func (c TestContext) Upgrade(nodes []Gravity, installerUrl, subdir string) error {
	ctx, cancel := context.WithTimeout(c.parent, withDuration(c.timeouts.Install, len(nodes)))
	defer cancel()

	if len(nodes) == 0 {
		return trace.Errorf("no nodes provided")
	}

	node := nodes[0]

	err := node.SetInstaller(ctx, installerUrl, subdir)
	if err != nil {
		return trace.Wrap(err)
	}

	err = node.Upgrade(ctx)
	return trace.Wrap(err)
}
