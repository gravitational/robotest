package gravity

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"time"

	"github.com/gravitational/robotest/lib/utils"

	"github.com/gravitational/trace"
)

const (
	localOpsCenterURL = `https://gravity-site.kube-system.svc.cluster.local:3009`
)

// Simple hook to allow re-entrance to already initialized host
func (c *TestContext) FromPreviousInstall(nodes []Gravity, subdir string) {
	for _, node := range nodes {
		g := node.(*gravity)
		g.installDir = filepath.Join(g.param.homeDir, subdir)
	}
}

// ProvisionInstaller deploys a specific installer
func (c *TestContext) SetInstaller(nodes []Gravity, installerUrl string, tag string) error {
	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Install)
	defer cancel()

	errs := make(chan error, len(nodes))
	for _, node := range nodes {
		go func(node Gravity) {
			errs <- node.SetInstaller(ctx, installerUrl, tag)
		}(node)
	}

	_, err := utils.Collect(ctx, cancel, errs, nil)
	if err = trace.Wrap(err); err != nil {
		return trace.Wrap(err)
	}

	// only forward node logs to the cloud
	if c.suite.client == nil {
		return nil
	}

	return nil
}

// OfflineInstall sets up cluster using nodes provided
func (c *TestContext) OfflineInstall(nodes []Gravity, param InstallParam) error {
	ctx, cancel := context.WithTimeout(c.parent, withDuration(c.timeouts.Install, len(nodes)))
	defer cancel()

	if len(nodes) == 0 {
		return trace.NotFound("at least one node")
	}

	master := nodes[0].(*gravity)
	if param.Token == "" {
		param.Token = "ROBOTEST"
	}
	if param.Cluster == "" {
		param.Cluster = master.param.Tag()
	}

	errs := make(chan error, len(nodes))
	go func() {
		errs <- master.Install(ctx, param)
	}()

	for _, node := range nodes[1:] {
		go func(n Gravity) {
			err := n.Join(ctx, JoinCmd{
				PeerAddr: master.Node().PrivateAddr(),
				Token:    param.Token,
				Role:     param.Role})
			if err != nil {
				n.Logger().WithError(err).Error("Join failed")
			}
			errs <- err
		}(node)
	}

	_, err := utils.Collect(ctx, cancel, errs, nil)
	if err != nil {
		c.Logger().WithError(err).Error("install failed")
		return trace.Wrap(err)
	}

	if param.EnableRemoteSupport {
		_, err = master.RunInPlanet(ctx, "/usr/bin/gravity",
			"site", "complete", "--support=on", "--insecure",
			fmt.Sprintf("--ops-url=%s", localOpsCenterURL),
			master.param.Tag())
	}

	return trace.Wrap(err)
}

func makePassword() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	const chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	result := make([]byte, 10)
	for i := range result {
		result[i] = chars[r.Intn(len(chars))]
	}
	return string(result)
}

// Uninstall makes nodes leave cluster and uninstall gravity
// it is not asserting internally
func (c *TestContext) Uninstall(nodes []Gravity) error {
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
func (c *TestContext) Upgrade(nodes []Gravity, installerUrl, subdir string) error {
	ctx, cancel := context.WithTimeout(c.parent, withDuration(2*c.timeouts.Install, len(nodes))) // DEBUG
	defer cancel()

	if len(nodes) == 0 {
		return trace.Errorf("no nodes provided")
	}

	node := nodes[0]

	err := node.SetInstaller(ctx, installerUrl, subdir)
	if err != nil {
		return trace.Wrap(err)
	}

	err = node.Upload(ctx)
	if err != nil {
		return trace.Wrap(err)
	}

	time.Sleep(3 * time.Minute) // DEBUG

	err = node.Upgrade(ctx)
	return trace.Wrap(err)
}
