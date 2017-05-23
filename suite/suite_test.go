package suite

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/robotest/suite/sanity"

	"github.com/stretchr/testify/assert"
)

var testSuite = flag.String("suite", "sanity", "test suite to run")
var configFile = flag.String("config", "", "cloud config file in YAML")
var stateDir = flag.String("dir", "", "state dir")
var tag = flag.String("tag", "", "tag to uniquely mark resources in cloud")

var testTimeout = time.Minute * 30

var suites = map[string]gravity.TestFunc{
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

	testFn, there := suites[*testSuite]
	if !there {
		t.Fatalf("No such test suite \"%s\"", *testSuite)
	}

	config := gravity.LoadConfig(t, *configFile, *stateDir, *tag)

	// testing package has internal 10 mins timeout, can be reset from command line only
	ctx, _ := context.WithTimeout(context.Background(), testTimeout)

	ok := t.Run(*testSuite, func(t *testing.T) {
		testFn(ctx, t, config, nil)
	})
	assert.True(t, ok, *testSuite)
}
