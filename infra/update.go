package infra

import (
	"context"

	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// UploadUpdate uploads a new application version to the installer node
func UploadUpdate(ctx context.Context, provisioner Provisioner, installer Node) (err error) {
	var session *ssh.Session
	err = wait.Retry(ctx, func() error {
		session, err = installer.Connect()
		if err != nil {
			log.Debug(trace.DebugReport(err))
		}
		return trace.Wrap(err)
	})
	if err != nil {
		errClose := session.Close()
		if errClose != nil {
			log.Errorf("Failed to close upload update SSH session: %v.", errClose)
		}
		return trace.Wrap(err)
	}
	defer session.Close()

	log.Debug("Start upload.")
	err = provisioner.UploadUpdate(session)
	if err != nil {
		return trace.Wrap(err)
	}

	return nil
}
