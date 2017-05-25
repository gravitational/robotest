package gravity

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/gravitational/trace"
)

// DestroyFn function which will destroy previously created remote resources
type DestroyFn func(context.Context, *testing.T) error

var destroyOnSuccess = flag.Bool("destroy-on-success", true, "remove resources after test success")
var destroyOnFailure = flag.Bool("destroy-on-failure", false, "remove resources after test failure")

var resourceListFile = flag.String("resourcegroupfile", "", "file with list of resources created")

// wrapDestroyFn implements a global conditional logic
func wrapDestroyFn(tag string, destroy func() error) DestroyFn {
	return func(ctx context.Context, t *testing.T) error {
		if (*destroyOnSuccess == false) ||
			(t.Failed() && *destroyOnFailure == false) {
			t.Logf("Not removing terraform %s")
			return nil
		}

		info := fmt.Sprintf("***\n*** destroying Terraform resources %s\n***\n", tag)
		t.Logf(info)
		log.Printf(info)

		err := destroy()
		if err != nil {
			t.Logf("%s : %v", info, err)
			log.Printf("%s : %v", info, err)
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
// as test might crash and leak resources on the cloud
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

	file, err := os.OpenFile(*resourceListFile, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return trace.Wrap(err, "updating resource allocation index %s : %v", *resourceListFile, err)
	}
	defer file.Close()

	for res, _ := range resourceAllocations.tags {
		log.Printf("currently allocated: %s", res)
		_, err = fmt.Fprintln(file, res)
		if err != nil {
			return trace.ConvertSystemError(err)
		}
	}

	return nil
}
