package suite

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/gravitational/robotest/infra/gravity"
	"github.com/stretchr/testify/require"
)

var configFile = flag.String("config", "", "cloud config file in YAML")
var stateDir = flag.String("dir", "", "state dir")
var tag = flag.String("tag", "", "tag to uniquely mark resources in cloud")

var testTimeout = time.Minute * 30

func TestBasic(t *testing.T) {
	config := gravity.LoadConfig(t, *configFile)

	logFn := gravity.Logf(t, "")

	var err error
	var dir = *stateDir
	if dir == "" {
		dir, err = ioutil.TempDir("", "robotest")
		require.NoError(t, err, "tmp dir")
	}
	logFn("State dir=%s", dir)

	runTag := *tag
	if runTag == "" {
		now := time.Now()
		runTag = fmt.Sprintf("robotest-%02d%02d-%02d%02d",
			now.Month(), now.Day(), now.Hour(), now.Minute())
	}

	// testing package has internal 10 mins timeout, can be reset from command line
	ctx, _ := context.WithTimeout(context.Background(), testTimeout)

	for _, os := range []string{"ubuntu"} {
		rmTag := fmt.Sprintf("%s-offline-basic-%s", runTag, os)
		t.Run(rmTag,
			func(t *testing.T) {
				t.Parallel()
				testOfflineBasic(ctx, t, os, config,
					rmTag, dir)
			})
	}
}
