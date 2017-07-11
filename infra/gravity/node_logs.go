package gravity

import (
	"context"
	"fmt"
	"path/filepath"

	sshutil "github.com/gravitational/robotest/lib/ssh"
)

func (g *gravity) streamLogs(ctx context.Context, file string) {
	path := filepath.Join(g.installDir, file)
	log := g.Logger().WithField("file_stream", path)

	err := sshutil.WaitForFile(ctx, g.Client(), log, path, sshutil.TestRegularFile)
	if err != nil {
		log.WithError(err).Error("error waiting for node logs")
		return
	}

	sshutil.Run(ctx, g.Client(), log,
		fmt.Sprintf("/usr/bin/tail -f %s", filepath.Join(g.installDir, file)), nil)
}
