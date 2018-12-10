package gravity

import (
	"context"

	sshutil "github.com/gravitational/robotest/lib/ssh"

	"github.com/gravitational/trace"
)

func (g *gravity) streamLogs(ctx context.Context) error {
	return trace.Wrap(sshutil.Run(ctx, g.Client(), g.Logger().WithField("source", "journalctl"),
		"sudo /bin/journalctl --follow --output=cat", nil))
}

func (g *gravity) streamStartupLogs(ctx context.Context) error {
	return trace.Wrap(sshutil.Run(ctx, g.Client(), g.Logger().WithField("source", "journalctl"),
		"sudo /bin/journalctl --identifier=startup-script --lines=all --output=cat --follow", nil))
}
