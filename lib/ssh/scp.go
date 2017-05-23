package sshutils

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gravitational/robotest/lib/utils"

	"github.com/gravitational/trace"

	"golang.org/x/crypto/ssh"
)

// CopyFile transfers local file to remote host directory
func PutFile(ctx context.Context, logFn utils.LogFnType, client *ssh.Client, srcPath, dstDir string) (remotePath string, err error) {
	session, err := client.NewSession()
	if err != nil {
		return "", trace.Wrap(err)
	}
	defer session.Close()

	f, err := os.Open(srcPath)
	if err != nil {
		return "", trace.Wrap(err)
	}
	defer f.Close()

	fi, err := os.Stat(srcPath)
	if err != nil {
		return "", trace.Wrap(err)
	}

	errCh := make(chan error, 2)
	go scpSendFile(session, f, fi, errCh)

	go func() {
		err := session.Run(fmt.Sprintf("/usr/bin/scp -tr %s", dstDir))
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGTERM)
		return "", trace.Errorf("scp timed out")
	case err := <-errCh:
		// either I/O goroutine returns error first, or command completes
		if err != nil {
			return "", trace.Wrap(err)
		}
		remotePath = filepath.Join(tmpDir, filepath.Base(srcPath))
		return remotePath, nil
	}
}

func scpSendFile(session *ssh.Session, file *os.File, fi os.FileInfo, errCh chan error) {
	cmd := fmt.Sprintf("C%04o %d %s\n", fi.Mode()&os.ModePerm, fi.Size(), fi.Name())

	out, err := session.StdinPipe()
	if err != nil {
		errCh <- trace.Wrap(err)
		return
	}
	defer out.Close()

	_, err = io.WriteString(out, cmd)
	if err != nil {
		errCh <- trace.Wrap(err)
		return
	}

	n, err := io.Copy(out, file)
	if err != nil {
		errCh <- trace.Wrap(err)
		return
	}
	if n != fi.Size() {
		errCh <- trace.Errorf("short write: %v %v", n, fi.Size())
		return
	}

	if _, err := out.Write([]byte{0x0}); err != nil {
		errCh <- trace.Wrap(err)
		return
	}
}
