package gravity

import (
	"context"

	sshutil "github.com/gravitational/robotest/lib/ssh"
)

func (g *gravity) streamLogs(ctx context.Context) {
	sshutil.Run(ctx, g.Client(), g.Logger().WithField("source", "journalctl"),
		"sudo /bin/journalctl -f -o cat", nil)
}
