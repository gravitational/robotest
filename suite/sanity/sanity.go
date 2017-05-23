package sanity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/infra/gravity"
)

// Basic are essential sanity tests
func Basic(ctx context.Context, t *testing.T, config *gravity.ProvisionerConfig, payload interface{}) {
	// all tests run in parallel
	for _, os := range []string{"ubuntu"} {
		for _, fn := range []gravity.TestFunc{
			basicResize,
			// installReliability,
		} {
			gravity.Run(ctx, t, config.WithOS(os), fn, nil, gravity.Parallel)
		}
	}
}
