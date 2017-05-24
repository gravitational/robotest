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

var testSuite = flag.String("suite", "sanity", "test suite to run")
var testSets = flag.String("set", "", "comma delimited test set out of suite to run, leave empty for all")
var configFile = flag.String("config", "", "cloud config file in YAML")
var stateDir = flag.String("dir", "", "state dir")
var tag = flag.String("tag", "", "tag to uniquely mark resources in cloud")
var osFlag = flag.String("os", "ubuntu", "comma delimited list of OS")

var testTimeout = time.Hour * 3

type testSet map[string]gravity.TestFunc

var suites = map[string]testSet{
	"sanity": sanity.Basic,
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

	osFlavors := []string{}
	for _, flavor := range strings.Split(*osFlag, ",") {
		osFlavors = append(osFlavors, flavor)
	}

	suiteSet, there := suites[*testSuite]
	if !there {
		t.Fatalf("No such test suite \"%s\"", *testSuite)
	}
	if *testSets != "" {
		baseSet := suiteSet
		suiteSet = map[string]gravity.TestFunc{}

		for _, set := range strings.Split(*testSets, ",") {
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
		for key, fn := range suiteSet {
			gravity.Run(ctx, t, config.WithTag(*testSuite).WithTag(key).WithOS(osFlavor), fn, gravity.Parallel)
		}
	}
}
