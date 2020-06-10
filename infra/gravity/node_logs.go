package gravity

import (
	"context"

	sshutil "github.com/gravitational/robotest/lib/ssh"

	"github.com/gravitational/trace"
)

func (g *gravity) streamLogs(ctx context.Context) error {
	// journalctl lives at /usr/bin/journalctl on SUSE and /bin/journalctl on RHEL, Ubuntu, etc.
	// Thus robotest must either rely on the journalctl from $PATH or plumb through os specific
	// config. -- walt 2020-06
	return trace.Wrap(sshutil.Run(ctx, g.Client(), g.Logger().WithField("source", "journalctl"),
		"sudo journalctl --follow --output=cat", nil))
}

func (g *gravity) streamStartupLogs(ctx context.Context) error {
	// journalctl lives at /usr/bin/journalctl on SUSE and /bin/journalctl on RHEL, Ubuntu, etc.
	// Thus robotest must either rely on the journalctl from $PATH or plumb through os specific
	// config. -- walt 2020-06
	return trace.Wrap(sshutil.Run(ctx, g.Client(), g.Logger().WithField("source", "journalctl"),
		"sudo journalctl --identifier=startup-script --lines=all --output=cat --follow", nil))
}
