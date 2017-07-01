package gravity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/terraform"
	"github.com/gravitational/robotest/lib/constants"
	"github.com/gravitational/robotest/lib/utils"

	"github.com/gravitational/trace"
)

// DestroyFn function which will destroy previously created remote resources
type DestroyFn func(context.Context, *testing.T) error

type ProvisionerPolicy struct {
	// DestroyOnSuccess instructs to remove any cloud resources after test completed OK
	DestroyOnSuccess bool
	// DestroyOnFailure instructs to cleanup any cloud resources after test completed with failure or context was timed out or interrupted
	DestroyOnFailure bool
	// AlwaysCollectLogs requests to fetch logs also from VMs where tests completed OK
	AlwaysCollectLogs bool
	// FailFast requests to interrupt all other tests when any of the tests failed
	FailFast bool
	// ResourceListFile keeps record of allocated and not cleaned up resources
	ResourceListFile string
	// CancelAllFn is top-level context cancellation function which is used to implement FailFast behaviour
	CancelAllFn func()
}

var policy ProvisionerPolicy

func SetProvisionerPolicy(p ProvisionerPolicy) {
	policy = p
}

var testStatus = map[bool]string{true: "failed", false: "ok"}

const finalTeardownTimeout = time.Minute * 5

// wrapDestroyFn implements a global conditional logic
func wrapDestroyFn(tag string, nodes []Gravity, destroy func(context.Context) error) DestroyFn {
	return func(baseContext context.Context, t *testing.T) error {
		log := utils.Logf(t, tag)

		defer func() {
			if r := recover(); r != nil {
				log("\n*****\n wrapDestroyFn %s PANIC %+v\n*****\n", tag, r)
			}
		}()

		skipLogCollection := false

		if baseContext.Err() != nil && policy.DestroyOnFailure == false {
			log("destroy skipped for %s: %v", tag, baseContext.Err())
			return trace.Wrap(baseContext.Err())
		}

		ctx := baseContext
		if baseContext.Err() != nil {
			log("main context %v, providing extra %v for %s teardown and cleanup",
				baseContext.Err(), finalTeardownTimeout, tag)
			skipLogCollection = true
			var cancel func()
			ctx, cancel = context.WithTimeout(context.Background(), finalTeardownTimeout)
			defer cancel()
		}

		if t.Failed() && policy.FailFast {
			log("test failed, FailFast=true requesting other tests teardown")
			defer policy.CancelAllFn()
		}

		if !skipLogCollection && (t.Failed() || policy.AlwaysCollectLogs) {
			log("collecting logs from nodes...")
			err := NewContext(ctx, t, DefaultTimeouts).CollectLogs("postmortem", nodes)
			if err != nil {
				log("warning: errors collecting logs : %v", err)
			}
		}

		if (policy.DestroyOnSuccess == false) ||
			(t.Failed() && policy.DestroyOnFailure == false) {
			log("Not removing terraform %s", tag)
			return nil
		}

		log("destroying Terraform resources %s, test %s", tag, testStatus[t.Failed()])

		err := destroy(ctx)
		if err != nil {
			log("error destroying terraform resources: %v", err)
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

// terraform deals with underlying terraform provisioner
func runTerraform(baseContext context.Context, baseConfig ProvisionerConfig, params cloudDynamicParams) ([]infra.Node, func(context.Context) error, error) {
	// there's an internal retry in provisioners,
	// however they get stuck sometimes and the only real way to deal with it is to kill and retry
	// as they'll pick up incomplete state from cloud and proceed
	// only second chance is provided
	//
	// TODO: this seems to require more thorough testing, and same approach applied to Destory
	//

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

	if policy.FailFast {
		policy.CancelAllFn()
	}

	return nil, nil, trace.NewAggregate(err, p.Destroy(baseContext))
}
