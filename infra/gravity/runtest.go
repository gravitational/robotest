package gravity

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestFunc func(ctx context.Context, t *testing.T, config *ProvisionerConfig)

const (
	Parallel   = true
	Sequential = false
)

// Run is a handy wrapper around test runner to have a properly assigned tag, state dir, etc.
func Run(ctx context.Context, t *testing.T, config *ProvisionerConfig, fn TestFunc, parallel bool) {
	ok := t.Run(config.Tag(), wrap(fn, ctx, config, parallel))
	if parallel {
		assert.True(t, ok, config.Tag())
	} else {
		require.True(t, ok, config.Tag())
	}
}

func wrap(fn TestFunc, ctx context.Context, cfg *ProvisionerConfig, parallel bool) func(*testing.T) {
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
	Install, Status, Uninstall, Leave time.Duration
}

var DefaultTimeouts = OpTimeouts{
	Install:   time.Minute * 10,
	Uninstall: time.Minute * 3,
	Status:    time.Second * 30,
	Leave:     time.Second * 30,
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
	require.NoError(c.t, err, msg)
}

func withDuration(d time.Duration, n int) time.Duration {
	return d * time.Duration(n)
}
