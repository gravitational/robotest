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
		return trace.Wrap(err, cmd)
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

// RunAndParse runs remote SSH command with environment variables set by `env`
// exitStatus is -1 if undefined
func RunAndParse(
	ctx context.Context,
	client *ssh.Client,
	log logrus.FieldLogger,
	cmd string,
	env map[string]string,
	parse OutputParseFn,
) (err error) {
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

	stdout, err := session.StdoutPipe()
	if err != nil {
		return trace.Wrap(err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return trace.Wrap(err)
	}

	log = log.WithField("cmd", cmd)

	errCh := make(chan error, 2)
	expectErrs := 1
	if parse != nil {
		expectErrs++
		go func() {
			err := parse(bufio.NewReader(
				&readLogger{log.WithField("stream", "stdout"), stdout}))
			errCh <- trace.Wrap(err)
		}()
	}

	go func() {
		log.Debug(cmd)
		errCh <- session.Run(fmt.Sprintf("%s %s", strings.Join(envStrings, " "), cmd))
	}()

	if parse != nil {
		go func() {
			r := bufio.NewReader(stderr)
			stderrLog := log.WithField("stream", "stderr")
			for {
				line, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if line != "" {
					stderrLog.Debug(line)
				}
			}
		}()
	}

	for i := 0; i < expectErrs; i++ {
		select {
		case <-ctx.Done():
			_ = session.Signal(ssh.SIGTERM)
			log.WithError(ctx.Err()).Debug("Context terminated, sent SIGTERM.")
			return trace.Wrap(ctx.Err())
		case err = <-errCh:
			switch sshError := err.(type) {
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
		*out = strings.Replace(string(b), `\r`, ``, -1)
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
