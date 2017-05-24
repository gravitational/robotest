package utils

import (
	"context"

	"github.com/gravitational/trace"
)

// WaitForErrors collects errors from channel provided, honouring timeout
// it will expect exactly cap(errChan) messages
func CollectErrors(ctx context.Context, errChan chan error) error {
	errors := []error{}
	for i := 0; i < cap(errChan); i++ {
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
