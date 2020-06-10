/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	"github.com/gravitational/robotest/lib/utils"

	"github.com/gravitational/trace"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// PutFile transfers local file to remote host directory
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

	errCh := make(chan error, 1)

	stdin, err := session.StdinPipe()
	if err != nil {
		return "", trace.Wrap(err)
	}

	go func() {
		const MiB = 1048576
		buf := make([]byte, 32*MiB)
		err := scpSendFile(stdin, f, fi, buf)
		stdin.Close()
		errCh <- trace.Wrap(err)
	}()

	errReceive := session.Run(fmt.Sprintf("/usr/bin/scp -tr %s", dstDir))
	errSend := utils.CollectErrors(ctx, errCh)
	err = trace.NewAggregate(errReceive, errSend)

	if ctx.Err() != nil {
		_ = session.Signal(ssh.SIGTERM)
		return "", trace.LimitExceeded("scp timed out")
	}

	if err != nil {
		if IsExitMissingError(err) {
			log.WithError(err).Warn("Session aborted unexpectedly (node destroyed?).")
		}
		return "", trace.Wrap(err)
	}

	remotePath = filepath.Join(dstDir, filepath.Base(srcPath))
	return remotePath, nil
}

func scpSendFile(out io.WriteCloser, file *os.File, fi os.FileInfo, buf []byte) error {
	_, err := fmt.Fprintf(out, "C%04o %d %s\n", fi.Mode()&os.ModePerm, fi.Size(), fi.Name())
	if err != nil {
		return trace.ConvertSystemError(err)
	}

	n, err := io.CopyBuffer(out, file, buf)
	if err != nil {
		return trace.ConvertSystemError(err)
	}
	if n != fi.Size() {
		return trace.BadParameter("short write: %v %v", n, fi.Size())
	}

	if _, err := out.Write([]byte{0x0}); err != nil {
		return trace.ConvertSystemError(err)
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
	errCh := make(chan error, 2)

	go func() {
		log.Debug(cmd)
		err := session.Run(cmd)
		errCh <- trace.Wrap(err)
	}()

	go func() {
		_, err := io.Copy(tgz, stdout)
		errCh <- trace.Wrap(err)
	}()

	go func() {
		_, _ = io.Copy(ioutil.Discard,
			&readLogger{log.WithField("stream", "stderr"), stderr})
	}()

	err = utils.CollectErrors(ctx, errCh)
	if ctx.Err() != nil {
		_ = session.Signal(ssh.SIGTERM)
		return trace.LimitExceeded("%s -> %s killed due to context cancelled", cmd, dst)
	}
	if err != nil {
		return trace.Wrap(err)
	}
	return nil
}
