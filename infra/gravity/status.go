package gravity

import (
	"context"
	"testing"

	"github.com/gravitational/robotest/lib/utils"

	"github.com/stretchr/testify/assert"
)

// Status walks around all nodes and checks whether they all feel OK
func Status(ctx context.Context, t *testing.T, nodes []Gravity) error {
	errs := make(chan error, len(nodes))

	for _, node := range nodes {
		go func(n Gravity) {
			status, err := n.Status(ctx)
			assert.NoError(t, err, n.String())
			t.Logf("%s status=%+v", n, status)

			errs <- err
		}(node)
	}

	return utils.CollectErrors(ctx, len(nodes), errs)
}
