package sshutils

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/gravitational/trace"

	"golang.org/x/crypto/ssh"
)

type OutputParseFn func(r *bufio.Reader) (out interface{}, err error)
type LogFnType func(format string, args ...interface{})

const exitStatusUndefined = -1

// RunAndParse runs remote SSH command with environment variables set by `env`
// exitStatus is -1 if undefined
func RunAndParse(ctx context.Context, logFn LogFnType, client *ssh.Client, cmd string, env map[string]string, parse OutputParseFn) (out interface{}, exitStatus int, err error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, exitStatusUndefined, trace.Wrap(err, "ssh session")
	}
	defer session.Close()

	if env != nil {
		for k, v := range env {
			if err := session.Setenv(k, v); err != nil {
				return nil, exitStatusUndefined, trace.Wrap(err)
			}
		}
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return nil, exitStatusUndefined, trace.Wrap(err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return nil, exitStatusUndefined, trace.Wrap(err)
	}

	go func() {
		r := bufio.NewReader(stderr)
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			logFn("%s [stderr] %s", cmd, line)
		}
	}()

	outCh := make(chan interface{}, 2)
	go func() {
		out, err := parse(bufio.NewReader(
			&readLogger{fmt.Sprintf("%s [stdout]", cmd), logFn, stdout}))
		if err != nil {
			outCh <- trace.Wrap(err)
		} else {
			outCh <- out
		}
	}()

	runCh := make(chan error, 2)
	go func() {
		runCh <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		return nil, exitStatusUndefined,
			trace.Errorf("%s timed out, sending SIGTERM: %v", cmd, session.Signal(ssh.SIGTERM))
	case err = <-runCh:
		if exitErr, isExitErr := err.(*ssh.ExitError); isExitErr {
			return nil, exitErr.ExitStatus(), nil
		}
		if err != nil {
			return nil, exitStatusUndefined, trace.Wrap(err)
		}
	}

	select {
	case <-ctx.Done():
		return nil, exitStatusUndefined, trace.Errorf("parse function timed out")
	case out := <-outCh:
		if outErr, isError := out.(error); isError {
			return nil, exitStatusUndefined, trace.Wrap(outErr)
		}
		return out, 0, nil
	}
}

func ParseDiscard(r *bufio.Reader) (interface{}, error) {
	io.Copy(ioutil.Discard, r)
	return nil, nil
}

type readLogger struct {
	prefix string
	logFn  LogFnType
	r      io.Reader
}

func (l *readLogger) Read(p []byte) (n int, err error) {
	n, err = l.r.Read(p)
	if err != nil {
		l.logFn("%s %q: %v", l.prefix, p[0:n], err)
	} else {
		l.logFn("%s %q", l.prefix, p[0:n])
	}
	return n, err
}
