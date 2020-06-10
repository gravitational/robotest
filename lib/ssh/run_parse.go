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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/gravitational/trace"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type OutputParseFn func(r *bufio.Reader) error

// Run is a simple method to run external program and don't care about its output or exit status
func Run(ctx context.Context, client *ssh.Client, log logrus.FieldLogger, cmd string, env map[string]string) error {
	err := RunAndParse(ctx, client, log, cmd, env, ParseDiscard)
	if err != nil {
		return trace.Wrap(err, "command %q failed", cmd)
	}

	return nil
}

const (
	term  = "unknown"
	termH = 40
	termW = 80
)

var (
	termModes = ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
)

// RunAndParse runs remote SSH command cmd with environment variables set with env.
// parse if set, will be provided the reader that consumes stdout of the command.
// Returns *ssh.ExitError if the command has completed with a non-0 exit code,
// *ssh.ExitMissingError if the other side has terminated the session without providing
// the exit code and nil for no errors
func RunAndParse(
	ctx context.Context,
	client *ssh.Client,
	log logrus.FieldLogger,
	cmd string,
	env map[string]string,
	parse OutputParseFn,
) (err error) {
	log = log.WithField("cmd", cmd)

	session, err := client.NewSession()
	if err != nil {
		return trace.Wrap(err)
	}
	defer session.Close()

	err = session.RequestPty(term, termH, termW, termModes)
	if err != nil {
		return trace.Wrap(err)
	}

	envStrings := []string{}
	for k, v := range env {
		envStrings = append(envStrings, fmt.Sprintf("%s=%s", k, v))
	}

	session.Stdin = new(bytes.Buffer)

	var stdout io.Reader
	if parse != nil {
		// Only create a pipe to remote command's stdout if it's going to be
		// processed, otherwise the remote command might block
		stdout, err = session.StdoutPipe()
		if err != nil {
			return trace.Wrap(err)
		}
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return trace.Wrap(err)
	}

	sessionCommand := fmt.Sprintf("%s %s", strings.Join(envStrings, " "), cmd)
	err = session.Start(sessionCommand)
	if err != nil {
		return trace.Wrap(err)
	}

	errCh := make(chan error, 2)
	expectErrors := 1
	if parse != nil {
		expectErrors = 2
		go func() {
			err := parse(bufio.NewReader(
				&readLogger{
					log: log.WithField("stream", "stdout"),
					r:   stdout,
				}))
			err = trace.Wrap(err)
			errCh <- err
		}()
	}

	go func() {
		r := bufio.NewReader(stderr)
		stderrLog := log.WithField("stream", "stderr")
		for {
			line, err := r.ReadString('\n')
			if line != "" {
				stderrLog.Debug(line)
			}
			if err != nil {
				return
			}
		}
	}()

	go func() {
		err := trace.Wrap(session.Wait())
		errCh <- err
	}()

	for i := 0; i < expectErrors; i++ {
		select {
		case <-ctx.Done():
			_ = session.Signal(ssh.SIGTERM)
			log.WithError(ctx.Err()).Debug("Context terminated, sent SIGTERM.")
			return trace.Wrap(ctx.Err())
		case err = <-errCh:
			switch sshError := trace.Unwrap(err).(type) {
			case *ssh.ExitError:
				err = trace.Wrap(sshError)
				log.WithError(err).Debugf("Command %v failed: %v", cmd, sshError.Error())
				return err
			case *ssh.ExitMissingError:
				err = trace.Wrap(sshError)
				log.WithError(err).Debug("Session aborted unexpectedly (node destroyed?).")
				return err
			}
			if err != nil {
				err = trace.Wrap(err)
				log.WithError(err).Debug("Unexpected error.")
				return err
			}
		}
	}

	return nil
}

func ParseDiscard(r *bufio.Reader) error {
	_, _ = io.Copy(ioutil.Discard, r)
	return nil
}

func ParseAsString(out *string) OutputParseFn {
	return func(r *bufio.Reader) error {
		b, err := ioutil.ReadAll(r)
		if err != nil {
			return trace.Wrap(err)
		}
		*out = string(bytes.TrimSpace(b))
		return nil
	}
}

func IsExitMissingError(err error) bool {
	_, ok := trace.Unwrap(err).(*ssh.ExitMissingError)
	return ok
}

// ExitStatusError describes the class of errors that
// report exit status
type ExitStatusError interface {
	// ExitStatus reports the exist status of an operation
	ExitStatus() int
}

type readLogger struct {
	log logrus.FieldLogger
	r   io.Reader
}

func (l *readLogger) Read(p []byte) (n int, err error) {
	n, err = l.r.Read(p)
	if err != nil && err != io.EOF {
		l.log.WithError(err).Debug("unexpected I/O error")
	} else if n > 0 {
		l.log.Debug(string(p[0:n]))
	}
	return n, err
}
