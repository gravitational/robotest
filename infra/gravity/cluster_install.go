package gravity

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gravitational/robotest/infra/providers/gce"
	"github.com/gravitational/robotest/lib/constants"
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
	// Cloud Provider ops will install telekube for us, so we can just exit early
	if c.provisionerCfg.CloudProvider == constants.Ops {
		return nil
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.WaitForInstaller)
	defer cancel()

	err := waitFileInstaller(ctx, installerUrl, c.Logger())
	if err != nil {
		return trace.Wrap(err)
	}

	ctx, cancel = context.WithTimeout(c.ctx, c.timeouts.Install)
	defer cancel()

	errs := make(chan error, len(nodes))
	for _, node := range nodes {
		go func(node Gravity) {
			errs <- node.SetInstaller(ctx, installerUrl, tag)
		}(node)
	}

	_, err = utils.Collect(ctx, cancel, errs, nil)
	return trace.Wrap(err)
}

// OfflineInstall sets up cluster using nodes provided
func (c *TestContext) OfflineInstall(nodes []Gravity, param InstallParam) error {
	// Cloud Provider ops will install telekube for us, so we can just exit early
	if c.provisionerCfg.CloudProvider == constants.Ops {
		return nil
	}

	c.Logger().Info("Offline install.")

	ctx, cancel := context.WithTimeout(c.ctx, withDuration(c.timeouts.Install, len(nodes)))
	defer cancel()

	param.CloudProvider = c.provisionerCfg.CloudProvider
	master := nodes[0].(*gravity)
	if param.Token == "" {
		param.Token = "ROBOTEST"
	}
	if param.Cluster == "" {
		param.Cluster = master.param.Tag()
	}
	if param.CloudProvider == constants.GCE {
		param.GCENodeTag = gce.TranslateClusterName(param.Cluster)
	}

	errs := make(chan error, len(nodes))
	go func() {
		c.Logger().WithField("node", master).Info("Install on leader node.")
		errs <- master.Install(ctx, param)
	}()

	for _, node := range nodes[1:] {
		go func(n Gravity) {
			c.Logger().WithField("node", n).Info("Join.")
			err := n.Join(ctx, JoinCmd{
				PeerAddr: master.Node().PrivateAddr(),
				Token:    param.Token,
				Role:     param.Role,
				StateDir: param.StateDir,
			})
			if err != nil {
				n.Logger().WithError(err).Warn("Join failed.")
			}
			errs <- err
		}(node)
	}

	_, err := utils.Collect(ctx, cancel, errs, nil)
	if err != nil {
		c.Logger().WithError(err).Warn("Install failed.")
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
			logger.Info("Waiting for installer file to become available.")
			return wait.Continue("waiting for installer file %s", file)
		}
		return wait.Abort(trace.ConvertSystemError(err))
	})

	return trace.Wrap(err)
}

// Uninstall makes nodes leave cluster and uninstall gravity
// it is not asserting internally
func (c *TestContext) Uninstall(nodes []Gravity) error {
	ctx, cancel := context.WithTimeout(c.ctx, withDuration(c.timeouts.Install, len(nodes)))
	defer cancel()

	errs := make(chan error, len(nodes))

	for _, node := range nodes {
		go func(n Gravity) {
			errs <- n.Uninstall(ctx)
		}(node)
	}

	return trace.Wrap(utils.CollectErrors(ctx, errs))
}

// UninstallApp uninstalls cluster application
func (c *TestContext) UninstallApp(nodes []Gravity) error {
	roles, err := c.NodesByRole(nodes)
	if err != nil {
		return trace.Wrap(err)
	}

	ctx, cancel := context.WithTimeout(context.TODO(), c.timeouts.UninstallApp)
	defer cancel()

	master := roles.ApiMaster
	err = master.UninstallApp(ctx)
	if err != nil {
		return trace.Wrap(err)
	}
	return nil
}

// Upgrade performs an upgrade procedure on all nodes
func (c *TestContext) Upgrade(nodes []Gravity, installerURL, gravityURL, subdir string) error {
	roles, err := c.NodesByRole(nodes)
	if err != nil {
		return trace.Wrap(err)
	}
	master := roles.ApiMaster
	err = c.uploadInstaller(roles.ApiMaster, roles.Other, installerURL, gravityURL, subdir)
	if err != nil {
		return trace.Wrap(err)
	}
	return c.upgrade(master, len(nodes))
}

func (c *TestContext) uploadInstaller(master Gravity, nodes []Gravity, installerURL, gravityURL, subdir string) error {
	log := c.Logger().WithField("leader", master)
	log.Info("Pull installer.")

	ctx, cancel := context.WithTimeout(c.ctx, withDuration(c.timeouts.Install, len(nodes)+1))
	defer cancel()

	err := master.SetInstaller(ctx, installerURL, subdir)
	if err != nil {
		return trace.Wrap(err)
	}

	err = c.Status(nodes)
	if err != nil {
		return trace.Wrap(err)
	}

	log.Info("Upload upgrade.")
	err = master.Upload(ctx)
	if err != nil {
		return trace.Wrap(err)
	}

	if len(nodes) > 1 {
		log.Info("Upload gravity binaries.")
		return uploadBinaries(ctx, nodes, gravityURL, subdir)
	}
	return nil
}

func (c *TestContext) upgrade(master Gravity, numNodes int) error {
	ctx, cancel := context.WithTimeout(c.ctx, withDuration(c.timeouts.Upgrade, numNodes))
	defer cancel()
	log.Info("Upgrade.")
	return master.Upgrade(ctx)
}

// ExecScript will run and execute a script on all nodes
func (c *TestContext) ExecScript(nodes []Gravity, scriptUrl string, args []string) error {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeouts.ExecScript)
	defer cancel()

	errs := make(chan error, len(nodes))
	for _, node := range nodes {
		go func(g Gravity) {
			errs <- trace.Wrap(g.ExecScript(ctx, scriptUrl, args))
		}(node)
	}

	return trace.Wrap(utils.CollectErrors(ctx, errs))
}

func uploadBinaries(ctx context.Context, nodes []Gravity, url, subdir string) error {
	errs := make(chan error, len(nodes))
	for _, node := range nodes {
		go func(node Gravity) {
			err := node.TransferFile(ctx, url, subdir)
			errs <- trace.Wrap(err)
		}(node)
	}
	return utils.CollectErrors(ctx, errs)
}
