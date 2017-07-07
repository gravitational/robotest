package wait

import (
	"context"
	"time"
)

// Sleep is context-interruptable sleep
func Sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
