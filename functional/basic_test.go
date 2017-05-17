package functional

import (
	"flag"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var configFile = flag.String("config", "", "cloud config file in YAML")
var stateDir = flag.String("config", "", "state dir")
var tag = flag.String("tag", "", "tag to uniquely mark resources in cloud")

func TestBasic(t *testing.T) {
	config, err := LoadConfig(*configFile)
	require.NoError(t, err, "config file")

	if *stateDir == "" {
		*stateDir, err = ioutil.TempDir("", "robotest")
		require.NoError(t, err, "tmp dir")
	}
	t.Logf("State dir=%s", *stateDir)

	if *tag == "" {
		now = time.Now()
		*tag = fmt.Sprintf("robotest-%02d%02d-%02d%02d",
			now.Month(), now.Day(), now.Hour(), now.Minute())
	}

	// testing package has internal 10 mins timeout, can be reset from command line
	ctx := context.WithTimeout(context.Background(), testTimeout)

	for _, os := range []string{"ubuntu"} {
		rmTag = fmt.Sprintf("offline-all-%s", os)
		t.Run(rmTag,
			func(t *testing.T) {
				t.Parallel()
				testOfflineAll(ctx, t, os, config,
					rmTag, *stateDir)
			})
	}
}
