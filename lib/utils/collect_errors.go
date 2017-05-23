package utils

import (
	"context"

	"github.com/gravitational/trace"
)

// WaitForErrors collects errors from channel provided, honouring timeout
func CollectErrors(ctx context.Context, count int, errChan chan error) error {
	if count <= 0 {
		return trace.Errorf("count(%d) <= 0", count)
	}

	errors := []error{}
	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			errors = append(errors, trace.Errorf("timed out"))
			break
		case err := <-errChan:
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	return trace.NewAggregate(errors...)
}
