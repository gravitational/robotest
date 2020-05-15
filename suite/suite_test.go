package suite

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/gravitational/robotest"
	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/robotest/lib/config"
	"github.com/gravitational/robotest/lib/debug"
	"github.com/gravitational/robotest/lib/defaults"
	"github.com/gravitational/robotest/lib/xlog"
	"github.com/gravitational/robotest/suite/sanity"

	log "github.com/sirupsen/logrus"
)

var testSuite = flag.String("suite", "sanity", "test suite to run")
var provision = flag.String("provision", "", "cloud credentials in JSON string")
var tag = flag.String("tag", "", "tag to uniquely mark resources in cloud")

var repeat = flag.Int("repeat", 1, "the number of times to schedule each test")
var retries = flag.Int("retries", defaults.MaxRetriesPerTest, "the number of times to retry a failed test")
var failFast = flag.Bool("fail-fast", false, "cancel all scheduled tests and retries on first failure")
var destroyOnSuccess = flag.Bool("destroy-on-success", true, "remove resources after test success")
var destroyOnFailure = flag.Bool("destroy-on-failure", false, "remove resources after test failure")

var resourceListFile = flag.String("resourcegroup-file", "", "file with list of resources created")
var collectLogs = flag.Bool("always-collect-logs", true, "collect logs from nodes once tests are finished. otherwise they will only be pulled for failed tests")

var cloudLogProjectID = flag.String("gcl-project-id", "", "enable logging to the cloud")

var debugFlag = flag.Bool("debug", false, "Verbose mode")
var debugPort = flag.Int("debug-port", 6060, "Profiling port")

var versionFlag = flag.Bool("version", false, "Display version information")

// max amount of time test will run
var testMaxTime = time.Hour * 12

var suites = map[string]*config.Config{
	"sanity": sanity.Suite(),
}

func setupSignals(suite gravity.TestSuite) {
	c := make(chan os.Signal, 3)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)

	go func() {
		for s := range c {
			suite.Logger().WithField("signal", s).Warn(s.String())
			suite.Cancel(s.String())
		}
	}()
}

// TestMain is a selector of which test to run,
// as go test cannot deal with multiple packages in pre-compiled mode
// right now it'll just invoke sanity suite
func TestMain(t *testing.T) {
	if *versionFlag {
		fmt.Printf("Version:\t%s\n", robotest.Version)
		fmt.Printf("Git Commit:\t%s\n", robotest.GitCommit)
		os.Exit(0)
	}

	if *testSuite == "" || *tag == "" {
		flag.Usage()
		t.Fatal("options required")
	}

	initLogger(*debugFlag)
	if *debugFlag {
		debug.StartProfiling(fmt.Sprintf("localhost:%v", *debugPort))
	}

	log.Debugf("Version:\t%s", robotest.Version)
	log.Debugf("Git Commit:\t%s", robotest.GitCommit)

	config := gravity.LoadConfig(t, []byte(*provision))
	config = config.WithTag(*tag)

	suiteCfg, there := suites[*testSuite]
	if !there {
		t.Fatalf("no such test suite %q", *testSuite)
	}

	testSet, err := suiteCfg.Parse(flag.Args())
	if err != nil {
		t.Fatalf("failed to parse args: %v", err)
	}

	// testing package has internal 10 mins timeout, can be reset from command line only
	// see docker/suite/entrypoint.sh
	ctx, cancel := context.WithTimeout(context.Background(), testMaxTime)
	defer cancel()

	policy := gravity.ProvisionerPolicy{
		DestroyOnSuccess:  *destroyOnSuccess,
		DestroyOnFailure:  *destroyOnFailure,
		AlwaysCollectLogs: *collectLogs,
		ResourceListFile:  *resourceListFile,
	}
	gravity.SetProvisionerPolicy(policy)

	logFields := log.Fields{
		"test_suite":         *testSuite,
		"test_set":           testSet,
		"provisioner_policy": policy,
		"tag":                *tag,
		"repeat":             *repeat,
		"fail_fast":          *failFast,
	}

	suite := gravity.NewSuite(ctx, t, *cloudLogProjectID, logFields, *failFast, *retries, defaults.MaxPreemptedRetriesPerTest)
	defer suite.Close()
	setupSignals(suite)

	for r := 1; r <= *repeat; r++ {
		for ts, entry := range testSet {
			suite.Schedule(entry.TestFunc,
				config.WithTag(fmt.Sprintf("%s-%d", ts, r)),
				entry.Param)
		}
	}

	result := suite.Run()
	logger := suite.Logger()
	for _, res := range result {
		logger.Debugf("%s %s %q %s", res.Name, res.Status, res.LogUrl, xlog.ToJSON(res.Param))
	}

	fmt.Println("\n******** TEST SUITE COMPLETED **********")
	for _, res := range result {
		fmt.Printf("%s %s %s %s\n", res.Status, res.Name, xlog.ToJSON(res.Param), res.LogUrl)
	}
}

func initLogger(debug bool) {
	level := log.InfoLevel
	if debug {
		level = log.DebugLevel
	}
	log.StandardLogger().Hooks = make(log.LevelHooks)
	log.SetOutput(os.Stderr)
	log.SetLevel(level)
}
