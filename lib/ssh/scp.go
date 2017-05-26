package sshutils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/gravitational/robotest/lib/constants"

	"github.com/gravitational/trace"

	"golang.org/x/crypto/ssh"
)

// CopyFile transfers local file to remote host directory
func PutFile(ctx context.Context, node SshNode, srcPath, dstDir string) (remotePath string, err error) {
	mkdirCmd := fmt.Sprintf("mkdir -p %s", dstDir)
	err = Run(ctx, node, mkdirCmd, nil)
	if err != nil {
		return "", trace.Wrap(err, "%v : %v", node, mkdirCmd)
	}
	session, err := node.Client().NewSession()
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
	go func() {
		err := scpSendFile(session, f, fi)
		errCh <- trace.Wrap(err)
	}()

	go func() {
		err := session.Run(fmt.Sprintf("/usr/bin/scp -tr %s", dstDir))
		errCh <- trace.Wrap(err)
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
		remotePath = filepath.Join(dstDir, filepath.Base(srcPath))
		return remotePath, nil
	}
}

func scpSendFile(session *ssh.Session, file *os.File, fi os.FileInfo) error {
	cmd := fmt.Sprintf("C%04o %d %s\n", fi.Mode()&os.ModePerm, fi.Size(), fi.Name())

	out, err := session.StdinPipe()
	if err != nil {
		return trace.Wrap(err)
	}
	defer out.Close()

	_, err = io.WriteString(out, cmd)
	if err != nil {
		return trace.Wrap(err)
	}

	n, err := io.Copy(out, file)
	if err != nil {
		return trace.Wrap(err)
	}
	if n != fi.Size() {
		return trace.Errorf("short write: %v %v", n, fi.Size())
	}

	if _, err := out.Write([]byte{0x0}); err != nil {
		return trace.Wrap(err)
	}

	return nil
}

// GetTgz will pick up number of remote files and store it locally as .tgz
// files should be absolute paths
func GetTgz(ctx context.Context, node SshNode, files []string, dst string) error {
	cmd := fmt.Sprintf("sudo tar cz -C / $(readlink -e %s)", strings.Join(files, " "))
	session, err := node.Client().NewSession()
	if err != nil {
		return trace.Wrap(err)
	}
	defer session.Close()

	session.Stdin = new(bytes.Buffer)

	stdout, err := session.StdoutPipe()
	if err != nil {
		return trace.ConvertSystemError(err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return trace.ConvertSystemError(err)
	}

	err = os.MkdirAll(filepath.Dir(dst), constants.SharedDirMask)
	if err != nil {
		return trace.ConvertSystemError(err)
	}
	tgz, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, constants.SharedReadMask)
	if err != nil {
		return trace.ConvertSystemError(err)
	}
	defer tgz.Close()

	errCh := make(chan error)

	go func() {
		node.Logf("(starting) %s", cmd)
		err := session.Run(cmd)
		if err != nil {
			if exitErr, isExitErr := err.(*ssh.ExitError); isExitErr {
				node.Logf("(exit=%d) %s", exitErr.ExitStatus(), cmd)
			} else {
				node.Logf("(error) %s, error=%v", err)
			}
		}
		errCh <- trace.Wrap(err)
	}()

	go func() {
		_, err := io.Copy(tgz, stdout)
		errCh <- trace.Wrap(err)
	}()

	go io.Copy(ioutil.Discard,
		&readLogger{fmt.Sprintf("%s [stderr]", cmd), node.Logf, stderr})

	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		return trace.Errorf("[%v] %s -> %s timed out", node, cmd, dst)
	case err = <-errCh:
		return trace.Wrap(err)
	}
	return nil
}
