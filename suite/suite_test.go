package suite

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/robotest/lib/config"
	"github.com/gravitational/robotest/lib/constants"
	"github.com/gravitational/robotest/lib/xlog"
	"github.com/gravitational/robotest/suite/sanity"

	"github.com/sirupsen/logrus"
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
var provision = flag.String("provision", "", "cloud credentials in JSON string")
var tag = flag.String("tag", "", "tag to uniquely mark resources in cloud")

var repeat = flag.Int("repeat", 1, "how many times to repeat a test")
var failFast = flag.Bool("fail-fast", false, "will attemt to shut down all other tests on first failure")
var destroyOnSuccess = flag.Bool("destroy-on-success", true, "remove resources after test success")
var destroyOnFailure = flag.Bool("destroy-on-failure", false, "remove resources after test failure")

var resourceListFile = flag.String("resourcegroup-file", "", "file with list of resources created")
var collectLogs = flag.Bool("always-collect-logs", true, "collect logs from nodes once tests are finished. otherwise they will only be pulled for failed tests")

var cloudLogProjectID = flag.String("gcl-project-id", "", "enable logging to the cloud")

var testSets valueList

// max amount of time test will run
var testMaxTime = time.Hour * 12

var suites = map[string]*config.Config{
	"sanity": sanity.Suite(),
}

func flavorSupported(os, version, storageDriver string) bool {
	if os != constants.OSRedHat {
		return true
	}

	if version == "7.4" {
		return true
	}

	if storageDriver == constants.DeviceMapper {
		return true
	}

	return false
}

func in(val string, arr []string) bool {
	for _, v := range arr {
		if val == v {
			return true
		}
	}
	return false
}

func setupSignals(suite gravity.TestSuite) {
	c := make(chan os.Signal, 3)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGHUP)
	signal.Notify(c, syscall.SIGINT)

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
	if *testSuite == "" || *tag == "" {
		flag.Usage()
		t.Fatal("options required")
	}

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
	ctx, cancelFn := context.WithTimeout(context.Background(), testMaxTime)
	defer cancelFn()

	policy := gravity.ProvisionerPolicy{
		DestroyOnSuccess:  *destroyOnSuccess,
		DestroyOnFailure:  *destroyOnFailure,
		AlwaysCollectLogs: *collectLogs,
		ResourceListFile:  *resourceListFile,
	}
	gravity.SetProvisionerPolicy(policy)

	suite := gravity.NewSuite(ctx, t, *cloudLogProjectID, logrus.Fields{
		"test_suite":         *testSuite,
		"test_set":           testSet,
		"provisioner_policy": policy,
		"tag":                *tag,
		"repeat":             *repeat,
		"fail_fast":          *failFast,
	}, *failFast)
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
	log := suite.Logger()
	for _, res := range result {
		log.Debugf("%s %s %q %s", res.Name, res.Status, res.LogUrl, xlog.ToJSON(res.Param))
	}

	fmt.Println("\n******** TEST SUITE COMPLETED **********")
	for _, res := range result {
		fmt.Printf("%s %s %s %s\n", res.Status, res.Name, xlog.ToJSON(res.Param), res.LogUrl)
	}
}
