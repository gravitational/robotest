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
var provision = flag.String("provision", "", "cloud credentials in JSON string")
var tag = flag.String("tag", "", "tag to uniquely mark resources in cloud")

var repeat = flag.Int("repeat", 1, "how many times to repeat a test")
var failFast = flag.Bool("fail-fast", false, "will attemt to shut down all other tests on first failure")
var destroyOnSuccess = flag.Bool("destroy-on-success", true, "remove resources after test success")
var destroyOnFailure = flag.Bool("destroy-on-failure", false, "remove resources after test failure")

var resourceListFile = flag.String("resourcegroup-file", "", "file with list of resources created")
var collectLogs = flag.Bool("always-collect-logs", true, "collect logs from nodes once tests are finished. otherwise they will only be pulled for failed tests")

var testSets, osFlavors, storageDrivers valueList

func init() {
	flag.Var(&osFlavors, "os", "comma delimited list of OS")
	flag.Var(&storageDrivers, "storage-driver", "comma delimited list of Docker storaga drivers: devicemapper,loopback,overlay,overlay2")
}

var testTimeout = time.Hour * 12

var suites = map[string]*config.Config{
	"sanity": sanity.Suite(),
}

var storageDriverOsCompat = map[string][]string{
	"ubuntu": []string{"overlay2", "overlay", "devicemapper", "loopback"},
	"debian": []string{"overlay2", "overlay", "devicemapper", "loopback"},
	"centos": []string{"overlay2", "overlay", "devicemapper", "loopback"},
	"rhel":   []string{"devicemapper", "loopback"},
}

func in(val string, arr []string) bool {
	for _, v := range arr {
		if val == v {
			return true
		}
	}
	return false
}

func setupSignals(cancelFn func()) {
	c := make(chan os.Signal, 3)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGHUP)
	signal.Notify(c, syscall.SIGINT)

	go func() {
		for s := range c {
			fmt.Println("GOT SIGNAL", s)
			cancelFn()
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

	config := gravity.ProvisionerConfig{}
	gravity.LoadConfig(t, []byte(*provision), &config)
	config = config.WithTag(*tag)

	t.Logf("CONFIG %q", config)

	suiteCfg, there := suites[*testSuite]
	if !there {
		t.Fatalf("No such test suite \"%s\"", *testSuite)
	}

	testSet, err := suiteCfg.Parse(flag.Args())
	if err != nil {
		t.Fatalf("failed to parse args: %v", err)
	}

	// testing package has internal 10 mins timeout, can be reset from command line only
	// see docker/suite/entrypoint.sh
	ctx, cancelFn := context.WithTimeout(context.Background(), testTimeout)
	setupSignals(cancelFn)

	gravity.SetProvisionerPolicy(gravity.ProvisionerPolicy{
		DestroyOnSuccess:  *destroyOnSuccess,
		DestroyOnFailure:  *destroyOnFailure,
		AlwaysCollectLogs: *collectLogs,
		FailFast:          *failFast,
		ResourceListFile:  *resourceListFile,
		CancelAllFn:       cancelFn,
	})

	t.Run(*testSuite, func(t *testing.T) {
		for r := 1; r <= *repeat; r++ {
			for _, osFlavor := range osFlavors {
				for ts, entry := range testSet {
					for _, drv := range storageDrivers {
						if in(drv, storageDriverOsCompat[osFlavor]) {
							gravity.Run(ctx, t,
								config.WithTag(fmt.Sprintf("%s-%d", ts, r)).WithOS(osFlavor).WithStorageDriver(drv),
								entry.TestFunc, gravity.Parallel)

							t.Logf("%s : %+v", ts, entry.Param)
						}
					}
				}
			}
		}
	})

	t.Logf("SUITE %s completed", *testSuite)
}
