package gravity

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"

	log "github.com/sirupsen/logrus"
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
	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.WaitForInstaller)
	err := waitFileInstaller(ctx, installerUrl, c.Logger())
	if err != nil {
		return trace.Wrap(err)
	}
	defer cancel()

	ctx, cancel = context.WithTimeout(c.parent, c.timeouts.Install)
	defer cancel()

	errs := make(chan error, len(nodes))
	for _, node := range nodes {
		go func(node Gravity) {
			errs <- node.SetInstaller(ctx, installerUrl, tag)
		}(node)
	}

	_, err = utils.Collect(ctx, cancel, errs, nil)
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
				Role:     param.Role,
				StateDir: param.StateDir,
			})
			if err != nil {
				n.Logger().WithError(err).Error("join failed")
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

func waitFileInstaller(ctx context.Context, file string, logger log.FieldLogger) error {
	u, err := url.Parse(file)
	if err != nil {
		return trace.Wrap(err, "parsing %s", file)
	}

	if u.Scheme != "" {
		return nil // real URL
	}

	err = wait.Retry(ctx, func() error {
		_, err := os.Stat(file)
		if err == nil {
			return nil
		}
		if os.IsNotExist(err) {
			return wait.Continue(fmt.Sprintf("waiting for installer file %s", file))
			logger.Warn("waiting for installer file to become available")
		}
		return wait.Abort(trace.ConvertSystemError(err))
	})

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
	roles, err := c.NodesByRole(nodes)
	if err != nil {
		return trace.Wrap(err)
	}

	master := roles.ApiMaster

	ctx, cancel := context.WithTimeout(c.parent, withDuration(c.timeouts.Install, len(nodes)))
	defer cancel()

	err = master.SetInstaller(ctx, installerUrl, subdir)
	if err != nil {
		return trace.Wrap(err)
	}

	err = master.Upload(ctx)
	if err != nil {
		return trace.Wrap(err)
	}

	ctx, cancel = context.WithTimeout(c.parent, withDuration(c.timeouts.Upgrade, len(nodes)))
	defer cancel()

	err = master.Upgrade(ctx)
	return trace.Wrap(err)
}

// ExecScript will run and execute a script on all nodes
func (c *TestContext) ExecScript(nodes []Gravity, scriptUrl string, args []string) error {
	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Status)
	defer cancel()

	errs := make(chan error, len(nodes))
	for _, node := range nodes {
		go func(g Gravity) {
			errs <- trace.Wrap(g.ExecScript(ctx, scriptUrl, args))
		}(node)
	}

	return trace.Wrap(utils.CollectErrors(ctx, errs))
}
