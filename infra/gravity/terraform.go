package gravity

import (
	"context"
	"flag"
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

var destroyOnSuccess = flag.Bool("destroy-on-success", true, "remove resources after test success")
var destroyOnFailure = flag.Bool("destroy-on-failure", false, "remove resources after test failure")

var resourceListFile = flag.String("resourcegroupfile", "", "file with list of resources created")

var testStatus = map[bool]string{true: "failed", false: "ok"}

// wrapDestroyFn implements a global conditional logic
func wrapDestroyFn(tag string, destroy func(context.Context) error) DestroyFn {
	return func(ctx context.Context, t *testing.T) error {
		log := utils.Logf(t, tag)

		if (*destroyOnSuccess == false) ||
			(t.Failed() && *destroyOnFailure == false) {
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
	if *resourceListFile == "" {
		return nil
	}

	file, err := os.OpenFile(*resourceListFile, os.O_RDWR|os.O_CREATE, constants.SharedReadMask)
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
func runTerraform(baseContext context.Context, baseConfig *ProvisionerConfig, params *cloudDynamicParams) ([]infra.Node, DestroyFn, error) {
	// there's an internal retry in provisioners,
	// however they get stuck sometimes and the only real way to deal with it is to kill and retry
	// as they'll pick up incomplete state from cloud and proceed
	// only second chance is provided
	for _, threshold := range []time.Duration{time.Second * 30, time.Minute * 5} {
		p, err := terraform.New(filepath.Join(baseConfig.stateDir, "tf"), params.tf)
		if err != nil {
			return nil, nil, trace.Wrap(err)
		}

		ctx, cancel := context.WithTimeout(baseContext, threshold)
		defer cancel()

		_, err = p.Create(ctx, false)
		if err != nil && ctx.Err() == context.DeadlineExceeded {
			continue
		}

		if err != nil {
			return nil, nil, trace.Wrap(err)
		}

		resourceAllocated(baseConfig.Tag())
		return p.NodePool().Nodes(), wrapDestroyFn(baseConfig.Tag(), p.Destroy), nil
	}

	return nil, nil, trace.Errorf("timed out provisioning %s", baseConfig.Tag())
}
