package suite

import (
	"context"
	"flag"
	"strings"
	"testing"
	"time"

	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/robotest/suite/sanity"
)

type valueList []string

func (r *valueList) String() string {
	if r == nil {
		return ""
	} else {
		return strings.Join(*r, ",")
	}
}
func (r *valueList) Set(value string) error {
	*r = strings.Split(value, ",")
	return nil
}

var testSuite = flag.String("suite", "sanity", "test suite to run")
var configFile = flag.String("config", "", "cloud config file in YAML")
var stateDir = flag.String("dir", "", "state dir")
var tag = flag.String("tag", "", "tag to uniquely mark resources in cloud")

var testSets, osFlavors, storageDrivers valueList

func init() {
	flag.Var(&testSets, "set", "comma delimited test set out of suite to run, leave empty for all")
	flag.Var(&osFlavors, "os", "comma delimited list of OS")
	flag.Var(&storageDrivers, "storage-driver", "comma delimited list of Docker storaga drivers, currently devicemapper or overlayfs2")
}

var testTimeout = time.Hour * 3

type testSet map[string]gravity.TestFunc

var suites = map[string]testSet{
	"sanity": sanity.Basic,
}

var storageDriverOsCompat = map[string][]string{
	"ubuntu": []string{"overlayfs2", "devicemapper"},
	"centos": []string{"overlayfs2", "devicemapper"},
	"rhel":   []string{"devicemapper"},
}

func in(val string, arr []string) bool {
	for _, v := range arr {
		if val == v {
			return true
		}
	}
	return false
}

// TestMain is a selector of which test to run,
// as go test cannot deal with multiple packages in pre-compiled mode
// right now it'll just invoke sanity suite
func TestMain(t *testing.T) {
	if *testSuite == "" || *configFile == "" || *stateDir == "" {
		flag.Usage()
		t.Fatal("options required")
	}

	config := gravity.LoadConfig(t, *configFile, *stateDir, *tag)

	suiteSet, there := suites[*testSuite]
	if !there {
		t.Fatalf("No such test suite \"%s\"", *testSuite)
	}
	if len(testSets) > 0 {
		baseSet := suiteSet
		suiteSet = map[string]gravity.TestFunc{}

		for _, set := range testSets {
			fn, there := baseSet[set]
			if !there {
				t.Fatalf("No such test set %s in suite %s", set, *testSuite)
			}
			suiteSet[set] = fn
		}
	}

	// testing package has internal 10 mins timeout, can be reset from command line only
	// see docker/suite/entrypoint.sh
	ctx, _ := context.WithTimeout(context.Background(), testTimeout)

	for _, osFlavor := range osFlavors {
		for ts, fn := range suiteSet {
			for _, drv := range storageDrivers {
				if in(drv, storageDriverOsCompat[osFlavor]) {
					gravity.Run(ctx, t,
						config.WithTag(*testSuite).WithTag(ts).WithOS(osFlavor).WithStorageDriver(drv),
						fn, gravity.Parallel)
				}
			}
		}
	}
}
