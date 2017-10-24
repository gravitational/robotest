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

const (
	exitStatusUndefined = -1
	exitCode            = "exit"
)

// Run is a simple method to run external program and don't care about its output or exit status
func Run(ctx context.Context, client *ssh.Client, log logrus.FieldLogger, cmd string, env map[string]string) error {
	exit, err := RunAndParse(ctx, client, log, cmd, env, ParseDiscard)
	if err != nil {
		return trace.Wrap(err, cmd)
	}

	if exit != 0 {
		return trace.Errorf("%s returned %d", cmd, exit)
	}

	return nil
}

const (
	term  = "xterm"
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
func RunAndParse(ctx context.Context, client *ssh.Client, log logrus.FieldLogger, cmd string, env map[string]string, parse OutputParseFn) (exitStatus int, err error) {
	session, err := client.NewSession()
	if err != nil {
		return exitStatusUndefined, trace.Wrap(err)
	}
	defer session.Close()

	err = session.RequestPty(term, termH, termW, termModes)
	if err != nil {
		return exitStatusUndefined, trace.Wrap(err)
	}

	envStrings := []string{}
	if env != nil {
		for k, v := range env {
			envStrings = append(envStrings, fmt.Sprintf("%s=%s", k, v))
		}
	}

	session.Stdin = new(bytes.Buffer)

	stdout, err := session.StdoutPipe()
	if err != nil {
		return exitStatusUndefined, trace.Wrap(err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return exitStatusUndefined, trace.Wrap(err)
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

	go func() {
		r := bufio.NewReader(stderr)
		stderrLog := log.WithField("stream", "stderr")
		for {
			line, err := r.ReadString('\n')
			if line != "" {
				stderrLog.Debug(line)
			}
			if parse == nil {
				session.Close()
				errCh <- nil // FIXME : this is a hack; session closure does not unblock session.Run() wonder if there's a better way
				return
			}
			if err != nil {
				return
			}
		}
	}()

	for i := 0; i < expectErrs; i++ {
		select {
		case <-ctx.Done():
			session.Signal(ssh.SIGTERM)
			log.WithError(ctx.Err()).Debug("context terminated, sent SIGTERM")
			return exitStatusUndefined, err
		case err = <-errCh:
			if exitErr, isExitErr := err.(*ssh.ExitError); isExitErr {
				err = trace.Wrap(exitErr)
				log.WithError(err).Debugf("%s : %s", cmd, exitErr.Error())
				return exitErr.ExitStatus(), err
			}
			if err != nil {
				err = trace.Wrap(err)
				log.WithError(err).Debug("unexpected error")
				return exitStatusUndefined, err
			}
		}
	}
	return 0, nil
}

func ParseDiscard(r *bufio.Reader) error {
	io.Copy(ioutil.Discard, r)
	return nil
}

func ParseAsString(out *string) OutputParseFn {
	return func(r *bufio.Reader) error {
		b, err := ioutil.ReadAll(r)
		if err != nil {
			return trace.Wrap(err)
		}
		*out = string(b)
		return nil
	}
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
