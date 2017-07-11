package sshutils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gravitational/robotest/lib/constants"

	"github.com/gravitational/trace"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// CopyFile transfers local file to remote host directory
func PutFile(ctx context.Context, client *ssh.Client, log logrus.FieldLogger, srcPath, dstDir string) (remotePath string, err error) {
	mkdirCmd := fmt.Sprintf("mkdir -p %s", dstDir)
	err = Run(ctx, client, log, mkdirCmd, nil)
	if err != nil {
		return "", trace.Wrap(err, mkdirCmd)
	}
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
	go func() {
		err := scpSendFile(session, f, fi)
		errCh <- trace.Wrap(err)
	}()

	go func() {
		err := session.Run(fmt.Sprintf("/usr/bin/scp -tr %s", dstDir))
		errCh <- trace.Wrap(err)
	}()

	for c := 0; c < 2; c++ {
		select {
		case <-ctx.Done():
			session.Signal(ssh.SIGTERM)
			return "", trace.Errorf("scp timed out")
		case err := <-errCh:
			if err != nil {
				return "", trace.Wrap(err)
			}
		}
	}
	remotePath = filepath.Join(dstDir, filepath.Base(srcPath))
	return remotePath, nil
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

// PipeCommand will run a remote command and store as local file
func PipeCommand(ctx context.Context, client *ssh.Client, log logrus.FieldLogger, cmd, dst string) error {
	session, err := client.NewSession()
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

	log = log.WithField("cmd", cmd)
	errCh := make(chan error, 3)

	go func() {
		log.Debug(cmd)
		err := session.Run(cmd)
		errCh <- trace.Wrap(err)
	}()

	go func() {
		_, err := io.Copy(tgz, stdout)
		errCh <- trace.Wrap(err)
	}()

	go io.Copy(ioutil.Discard,
		&readLogger{log.WithField("stream", "stderr"), stderr})

	select {
	case <-ctx.Done():
		err = trace.Errorf("%s -> %s killed due to context cancelled", cmd, dst)
		log.WithError(err).Error(err.Error())
		session.Signal(ssh.SIGKILL)
		return err
	case err = <-errCh:
		err = trace.Wrap(err)
		if err != nil {
			log.WithError(err).Errorf("%s: %s", cmd, err.Error())
		}
		return err
	}
	return nil
}
