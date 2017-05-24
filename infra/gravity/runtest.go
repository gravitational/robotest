package gravity

import (
	"context"
	"testing"

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
