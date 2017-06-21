package gravity

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestFunc func(ctx context.Context, t *testing.T, config ProvisionerConfig)

const (
	Parallel   = true
	Sequential = false
)

// Run is a handy wrapper around test runner to have a properly assigned tag, state dir, etc.
func Run(ctx context.Context, t *testing.T, config ProvisionerConfig, fn TestFunc, parallel bool) {
	ok := t.Run(config.Tag(), wrap(fn, ctx, config, parallel))
	if parallel {
		assert.True(t, ok, config.Tag())
	} else {
		require.True(t, ok, config.Tag())
	}
}

func wrap(fn TestFunc, ctx context.Context, cfg ProvisionerConfig, parallel bool) func(*testing.T) {
	return func(t *testing.T) {
		if parallel {
			t.Parallel()
		}
		fn(ctx, t, cfg)
	}
}

// OpTimeouts define per-node, per-operation timeouts which would be used to determine
// whether test must be failed
// provisioner has its own timeout / restart logic which is dependant on cloud provider and terraform
type OpTimeouts struct {
	Install, Status, Uninstall, Leave, CollectLogs time.Duration
}

var DefaultTimeouts = OpTimeouts{
	Install:     time.Minute * 15,
	Uninstall:   time.Minute * 5,
	Status:      time.Minute * 3,
	Leave:       time.Minute * 40,
	CollectLogs: time.Minute * 7,
}

// TestContext aggregates common parameters for better test suite readability
type TestContext struct {
	t        *testing.T
	parent   context.Context
	timeouts OpTimeouts
}

// NewContext creates new context holder
func NewContext(parent context.Context, t *testing.T, timeouts OpTimeouts) TestContext {
	return TestContext{t, parent, timeouts}
}

// OK is equivalent to require.NoError
func (c TestContext) OK(msg string, err error) {
	require.NoError(c.t, err, fmt.Sprintf("%s : %v", msg, err))
	if err == nil {
		c.Logf("*** %s: OK!", msg)
	}
}

// Sleep will just sleep with log message
func (c TestContext) Sleep(msg string, d time.Duration) {
	c.Logf("Sleep %v %s...", d, msg)
	select {
	case <-time.After(d):
	case <-c.parent.Done():
	}
}

func (c TestContext) Logf(format string, args ...interface{}) {
	c.t.Logf(format, args...)
	log.Printf("%s %s", c.t.Name(), fmt.Sprintf(format, args...))
}

func withDuration(d time.Duration, n int) time.Duration {
	return d * time.Duration(n)
}
