package gravity

import (
	"context"

	"github.com/gravitational/robotest/lib/utils"

	"github.com/stretchr/testify/assert"
)

// Status walks around all nodes and checks whether they all feel OK
func (c TestContext) Status(nodes []Gravity) error {
	ctx, cancel := context.WithTimeout(c.parent, c.timeouts.Status)
	defer cancel()

	errs := make(chan error, len(nodes))

	for _, node := range nodes {
		go func(n Gravity) {
			status, err := n.Status(ctx)
			assert.NoError(c.t, err, n.String())
			c.t.Logf("%s status=%+v", n, status)

			errs <- err
		}(node)
	}

	return utils.CollectErrors(ctx, errs)
}
