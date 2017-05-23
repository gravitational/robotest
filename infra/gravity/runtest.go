package gravity

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestFunc func(ctx context.Context, t *testing.T, config *ProvisionerConfig, payload interface{})

const (
	Parallel   = true
	Sequential = false
)

var reFuncName = regexp.MustCompile(`^([\w\/\.]+)\/([\d\w]+)\.([\d\w]+)$`)

// Run is a handy wrapper around test runner to have a properly assigned tag, state dir, etc.
func Run(ctx context.Context, t *testing.T, config *ProvisionerConfig, fn TestFunc, payload interface{}, parallel bool) bool {
	fnName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	vars := reFuncName.FindStringSubmatch(fnName)
	require.Len(t, vars, 4, "internal error")

	cfg := config.WithTag(fmt.Sprintf("%s-%s", vars[2], vars[3]))

	ok := t.Run(cfg.Tag(), wrap(fn, ctx, cfg, payload, parallel))
	require.True(t, ok, cfg.Tag())
	return ok
}

func wrap(fn TestFunc, ctx context.Context, cfg *ProvisionerConfig, payload interface{}, parallel bool) func(*testing.T) {
	return func(t *testing.T) {
		if parallel {
			t.Parallel()
		}
		fn(ctx, t, cfg, payload)
	}
}
