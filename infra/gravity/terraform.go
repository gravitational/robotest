package gravity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/providers/gce"
	"github.com/gravitational/robotest/infra/terraform"
	"github.com/gravitational/robotest/lib/constants"
	"github.com/gravitational/robotest/lib/defaults"
	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
)

// DestroyFn function which will destroy previously created remote resources
type DestroyFn func() error

type ProvisionerPolicy struct {
	// DestroyOnSuccess instructs to remove any cloud resources after test completed OK
	DestroyOnSuccess bool
	// DestroyOnFailure instructs to cleanup any cloud resources after test completed with failure or context was timed out or interrupted
	DestroyOnFailure bool
	// AlwaysCollectLogs requests to fetch logs also from VMs where tests completed OK
	AlwaysCollectLogs bool
	// ResourceListFile keeps record of allocated and not cleaned up resources
	ResourceListFile string
}

var policy ProvisionerPolicy

func SetProvisionerPolicy(p ProvisionerPolicy) {
	policy = p
}

var testStatus = map[bool]string{true: "failed", false: "ok"}

const finalTeardownTimeout = time.Minute * 5

// wrapDestroyFunc returns a function that wraps the specified set of nodes
// and the given clean up function that implements report collection and resource clean up.
func wrapDestroyFunc(c *TestContext, tag string, nodes []Gravity, destroy func(context.Context) error) DestroyFn {
	return func() error {
		defer func() {
			if r := recover(); r != nil {
				c.Logger().WithFields(
					logrus.Fields{
						"stack": string(debug.Stack()),
						"where": r,
					},
				).Error("Panic in terraform destroy.")
			}
		}()

		log := c.Logger().WithFields(logrus.Fields{
			"nodes":              nodes,
			"provisioner_policy": policy,
			"test_status":        testStatus[c.Failed()],
		})

		skipLogCollection := false
		ctx := c.Context()

		if ctx.Err() != nil && !policy.DestroyOnFailure {
			log.WithError(ctx.Err()).Info("Skipping destroy.")
			return trace.Wrap(ctx.Err())
		}

		if ctx.Err() != nil {
			skipLogCollection = true
		}

		if !skipLogCollection && (c.Failed() || policy.AlwaysCollectLogs) {
			log.Debug("Collecting logs from nodes...")
			err := c.CollectLogs("postmortem", nodes)
			if err != nil {
				log.WithError(err).Warn("Failed to collect node logs.")
			}
		}

		if !policy.DestroyOnSuccess ||
			(c.Failed() && !policy.DestroyOnFailure) {
			log.Info("not destroying VMs per policy")
			return nil
		}

		// Close the monitor processes
		c.monitorCancel()

		log.Info("Destroying VMs.")

		err := destroyResource(destroy)
		if err != nil {
			log.WithError(err).Error("Failed to destroy VM resources.")
		} else {
			if errDestroy := resourceDestroyed(tag); errDestroy != nil {
				log.WithError(errDestroy).Warn("Failed to remove resource account.")
			}
		}

		return trace.Wrap(err)
	}
}

var resourceAllocations = struct {
	sync.Mutex
	tags map[string]bool
}{tags: map[string]bool{}}

// resourceAllocated adds resource allocated into local index file for shell-based cleanup
// as test might crash and leak resources in the cloud
func resourceAllocated(tag string) error {
	resourceAllocations.Lock()
	defer resourceAllocations.Unlock()

	if _, there := resourceAllocations.tags[tag]; there {
		return trace.Errorf("resource tag not unique : %s", tag)
	}

	resourceAllocations.tags[tag] = true
	return saveResourceAllocations()
}

func resourceDestroyed(tag string) error {
	resourceAllocations.Lock()
	defer resourceAllocations.Unlock()

	delete(resourceAllocations.tags, tag)
	return saveResourceAllocations()
}

func saveResourceAllocations() error {
	if policy.ResourceListFile == "" {
		return nil
	}

	file, err := os.OpenFile(policy.ResourceListFile, os.O_RDWR|os.O_CREATE, constants.SharedReadMask)
	if err != nil {
		return trace.ConvertSystemError(err)
	}
	defer file.Close()

	for res := range resourceAllocations.tags {
		_, err = fmt.Fprintln(file, res)
		if err != nil {
			return trace.ConvertSystemError(err)
		}
	}

	return nil
}

// makeDynamicParams takes base config, validates it and returns cloudDynamicParams
func makeDynamicParams(baseConfig ProvisionerConfig) (*cloudDynamicParams, error) {
	param := cloudDynamicParams{ProvisionerConfig: baseConfig}

	// OS name is cloud-init script specific
	// enforce compatible values
	var ok bool
	usernames := map[string]map[string]string{
		constants.Azure: {
			"ubuntu": "robotest",
			"debian": "admin",
			"redhat": "redhat",
			"centos": "centos",
			"suse":   "robotest",
		},
		constants.GCE: {
			"ubuntu": "ubuntu",
			"debian": "robotest",
			"redhat": "redhat",
			"centos": "centos",
			"suse":   "robotest",
		},
		constants.AWS: {
			"ubuntu": "ubuntu",
			"debian": "admin",
			"redhat": "redhat",
			"centos": "centos",
		},
		constants.Ops: {
			"centos": "centos",
		},
	}

	param.user, ok = usernames[baseConfig.CloudProvider][baseConfig.os.Vendor]
	if !ok {
		return nil, trace.BadParameter("unknown OS vendor: %q", baseConfig.os.Vendor)
	}

	param.homeDir = filepath.Join("/home", param.user)

	param.terraform = terraform.Config{
		CloudProvider: baseConfig.CloudProvider,
		ScriptPath:    baseConfig.ScriptPath,
		NumNodes:      int(baseConfig.NodeCount),
		OS:            baseConfig.os.String(),
	}

	if baseConfig.AWS != nil {
		// AWS configuration is also used to download from S3 (i.e. even with
		// another cloud provider configured)
		config := *baseConfig.AWS
		param.terraform.AWS = &config
		param.terraform.AWS.ClusterName = baseConfig.tag
		param.terraform.AWS.SSHUser = param.user
		param.env = map[string]string{
			"AWS_ACCESS_KEY_ID":     param.terraform.AWS.AccessKey,
			"AWS_SECRET_ACCESS_KEY": param.terraform.AWS.SecretKey,
			"AWS_DEFAULT_REGION":    param.terraform.AWS.Region,
		}
	}

	switch {
	case baseConfig.Azure != nil:
		config := *baseConfig.Azure
		param.terraform.Azure = &config
		param.terraform.Azure.ResourceGroup = baseConfig.tag
		param.terraform.Azure.SSHUser = param.user
		param.terraform.Azure.Location = baseConfig.cloudRegions.Next()
	case baseConfig.GCE != nil:
		config := *baseConfig.GCE
		param.terraform.GCE = &config
		param.terraform.GCE.SSHUser = param.user
		param.terraform.GCE.Region = baseConfig.cloudRegions.Next()
		param.terraform.GCE.NodeTag = gce.TranslateClusterName(baseConfig.tag)
		param.terraform.VarFilePath = baseConfig.GCE.VarFilePath
	}

	return &param, nil
}

func runTerraform(ctx context.Context, baseConfig ProvisionerConfig, logger logrus.FieldLogger) (resp *terraformResp, err error) {
	retryer := wait.Retryer{
		Delay:       defaults.TerraformRetryDelay,
		Attempts:    defaults.TerraformRetries,
		FieldLogger: logger,
	}

	retry := 0
	cfg := baseConfig
	err = retryer.Do(ctx, func() error {
		if retry != 0 {
			cfg = baseConfig.WithTag(fmt.Sprintf("R%d", retry))
			logger.WithFields(logrus.Fields{
				"state-dir": cfg.StateDir,
				"tag":       cfg.Tag(),
			}).Info("Retrying terraform provisioning.")
		}
		retry++

		params, err := makeDynamicParams(cfg)
		if err != nil {
			return wait.Abort(trace.Wrap(err))
		}

		resp, err = runTerraformOnce(ctx, cfg, *params, logger)
		if err == nil {
			return nil
		}

		logger.WithError(err).Warn("terraform provisioning failed")
		return wait.Continue(err.Error())
	})
	return resp, trace.Wrap(err)
}

// terraform deals with underlying terraform provisioner
func runTerraformOnce(
	baseContext context.Context,
	baseConfig ProvisionerConfig,
	params cloudDynamicParams,
	logger logrus.FieldLogger,
) (resp *terraformResp, err error) {
	// there's an internal retry in provisioners,
	// however they get stuck sometimes and the only real way to deal with it is to kill and retry
	// as they'll pick up incomplete state from cloud and proceed
	// only second chance is provided
	//
	// TODO: this seems to require more thorough testing, and same approach applied to Destroy
	p, err := terraform.New(filepath.Join(baseConfig.StateDir, "tf"), params.terraform)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	for _, threshold := range []time.Duration{time.Minute * 15, time.Minute * 10} {
		ctx, cancel := context.WithTimeout(baseContext, threshold)
		defer cancel()

		_, err = p.Create(ctx, false)
		if ctx.Err() != nil {
			teardownCtx, cancel := context.WithTimeout(context.Background(), finalTeardownTimeout)
			defer cancel()
			err1 := trace.Errorf("[terraform interrupted on apply due to upper context=%v, result=%v]", ctx.Err(), err)
			err2 := trace.Wrap(p.Destroy(teardownCtx))
			return nil, trace.NewAggregate(err1, err2)
		}

		if err != nil {
			continue
		}

		if errAlloc := resourceAllocated(baseConfig.Tag()); errAlloc != nil {
			logger.Warnf("Failed to account for resource allocation: %v.", errAlloc)
		}

		return &terraformResp{
			nodes:     p.NodePool().Nodes(),
			destroyFn: p.Destroy,
			params:    params,
		}, nil
	}

	return nil, trace.NewAggregate(err, p.Destroy(baseContext))
}

// terraformResp describes the result of provisioning infrastructure with terraform.
type terraformResp struct {
	nodes     []infra.Node
	destroyFn func(context.Context) error
	params    cloudDynamicParams
}
