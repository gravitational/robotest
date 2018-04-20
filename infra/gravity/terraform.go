package gravity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gravitational/robotest/infra"
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

// wrapDestroyFn implements a global conditional logic
func wrapDestroyFn(c *TestContext, tag string, nodes []Gravity, destroy func(context.Context) error) DestroyFn {
	return func() error {
		defer func() {
			if r := recover(); r != nil {
				c.Logger().WithField("panic", r).Error("panic in terraform destroy")
			}
		}()

		log := c.Logger().WithFields(logrus.Fields{
			"nodes":              nodes,
			"provisioner_policy": policy,
			"test_status":        testStatus[c.Failed()],
		})

		skipLogCollection := false
		ctx := c.Context()

		if ctx.Err() != nil && policy.DestroyOnFailure == false {
			log.WithError(ctx.Err()).Info("skipped destroy")
			return trace.Wrap(ctx.Err())
		}

		if ctx.Err() != nil {
			log.WithError(ctx.Err()).Warn("extra cycles for teardown")
			skipLogCollection = true
			var cancel func()
			ctx, cancel = context.WithTimeout(context.Background(), finalTeardownTimeout)
			defer cancel()
		}

		if !skipLogCollection && (c.Failed() || policy.AlwaysCollectLogs) {
			log.Debug("collecting logs from nodes...")
			err := c.CollectLogs("postmortem", nodes)
			if err != nil {
				log.WithError(err).Error("collecting logs")
			}
		}

		if (policy.DestroyOnSuccess == false) ||
			(c.Failed() && policy.DestroyOnFailure == false) {
			log.Info("not destroying VMs per policy")
			return nil
		}

		log.Info("destroying VMs")

		err := destroyResource(destroy)
		if err != nil {
			log.WithError(err).Error("destroying VM resources")
		} else {
			resourceDestroyed(tag)
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

	for res, _ := range resourceAllocations.tags {
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
		"azure": map[string]string{
			"ubuntu": "robotest",
			"debian": "admin",
			"redhat": "redhat",
			"centos": "centos",
			"suse":   "robotest",
		},
		// TODO: verify this (copy of azure)
		"gce": map[string]string{
			"ubuntu": "robotest",
			"debian": "admin",
			"redhat": "redhat",
			"centos": "centos",
			"suse":   "robotest",
		},
		"aws": map[string]string{
			"ubuntu": "ubuntu",
			"debian": "admin",
			"redhat": "redhat",
			"centos": "centos",
		},
		"ops": map[string]string{
			"centos": "centos",
		},
	}

	param.user, ok = usernames[baseConfig.CloudProvider][baseConfig.os.Vendor]
	if !ok {
		return nil, trace.BadParameter("unknown OS vendor: %q", baseConfig.os.Vendor)
	}

	param.homeDir = filepath.Join("/home", param.user)

	param.tf = terraform.Config{
		CloudProvider: baseConfig.CloudProvider,
		ScriptPath:    baseConfig.ScriptPath,
		NumNodes:      int(baseConfig.NodeCount),
		OS:            baseConfig.os.String(),
	}

	if baseConfig.AWS != nil {
		aws := *baseConfig.AWS
		param.tf.AWS = &aws
		param.tf.AWS.ClusterName = baseConfig.tag
		param.tf.AWS.SSHUser = param.user

		param.env = map[string]string{
			"AWS_ACCESS_KEY_ID":     param.tf.AWS.AccessKey,
			"AWS_SECRET_ACCESS_KEY": param.tf.AWS.SecretKey,
			"AWS_DEFAULT_REGION":    param.tf.AWS.Region,
		}
	}

	if baseConfig.Azure != nil {
		azure := *baseConfig.Azure
		param.tf.Azure = &azure
		param.tf.Azure.ResourceGroup = baseConfig.tag
		param.tf.Azure.SSHUser = param.user
		param.tf.Azure.Location = cloudRegions.Next()
	}

	if baseConfig.GCE != nil {
		config := *baseConfig.GCE
		param.tf.GCE = &config
		param.tf.GCE.ClusterName = fmt.Sprintf("robotest-%v", baseConfig.tag)
		param.tf.GCE.SSHUser = param.user
		// FIXME: rotate regions?
		param.tf.GCE.Region = cloudRegions.Next()
	}

	return &param, nil
}

func runTerraform(ctx context.Context, baseConfig ProvisionerConfig, logger logrus.FieldLogger) (resp *terraformResp, err error) {
	retr := wait.Retryer{
		Delay:       defaults.TerraformRetryDelay,
		Attempts:    defaults.TerraformRetries,
		FieldLogger: logger,
	}

	retry := 0
	cfg := baseConfig
	resp = &terraformResp{}

	err = retr.Do(ctx, func() error {
		if retry != 0 {
			cfg = baseConfig.WithTag(fmt.Sprintf("R%d", retry))
			logger.Info("retrying terraform provisioning")
		}
		retry++

		resp.params, err = makeDynamicParams(cfg)
		if err != nil {
			return wait.Abort(trace.Wrap(err))
		}
		resp.nodes, resp.destroyFn, err = runTerraformOnce(ctx, cfg, *params)

		if err == nil {
			return nil
		}

		logger.WithError(err).Warn("terraform provisioning failed")
		return wait.Continue(err.Error())
	})

	if err == nil {
		return resp, nil
	}
	return nil, trace.Wrap(err)
}

// terraform deals with underlying terraform provisioner
func runTerraformOnce(baseContext context.Context, baseConfig ProvisionerConfig, params cloudDynamicParams) ([]infra.Node, DestroyFn, error) {
	// there's an internal retry in provisioners,
	// however they get stuck sometimes and the only real way to deal with it is to kill and retry
	// as they'll pick up incomplete state from cloud and proceed
	// only second chance is provided
	//
	// TODO: this seems to require more thorough testing, and same approach applied to Destory
	p, err := terraform.New(filepath.Join(baseConfig.StateDir, "tf"), params.tf)
	if err != nil {
		return nil, nil, trace.Wrap(err)
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
			return nil, nil, trace.NewAggregate(err1, err2)
		}

		if err != nil {
			continue
		}

		resourceAllocated(baseConfig.Tag())
		return p.NodePool().Nodes(), p.Destroy, nil
	}

	return nil, nil, trace.NewAggregate(err, p.Destroy(baseContext))
}

type terraformResp struct {
	nodes     []infra.Node
	destroyFn DestroyFn
	params    *cloudDynamicParams
}
