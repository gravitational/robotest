package sshutils

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"github.com/gravitational/trace"

	"golang.org/x/crypto/ssh"
)

type outputParseFn func(r *bufio.Reader) (out interface{}, err error)
type LogFnType func(format string, args ...interface{})

// RunAndParse runs remote SSH command with environment variables set by `env`
func RunAndParse(ctx context.Context, logFn LogFnType, session *ssh.Session, cmd string, env map[string]string, parse outputParseFn) (interface{}, error) {
	defer session.Close()

	if env != nil {
		for k, v := range env {
			if err := session.Setenv(k, v); err != nil {
				return nil, trace.Wrap(err)
			}
		}
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return nil, trace.Wrap(err)
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

	runCh := make(chan error)
	go func() {
		runCh <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		return nil, trace.Wrap(session.Signal(ssh.SIGTERM), "%s timed out", cmd)
	case err = <-runCh:
		if err != nil {
			return nil, trace.Wrap(err)
		}
	}

	select {
	case <-ctx.Done():
		return nil, trace.Errorf("parse function timed out")
	case out := <-outCh:
		if outErr, isError := out.(error); isError {
			return nil, trace.Wrap(outErr)
		}
		return out, nil
	}
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
