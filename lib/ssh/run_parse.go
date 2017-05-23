package sshutils

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/gravitational/robotest/lib/utils"

	"github.com/gravitational/trace"

	"golang.org/x/crypto/ssh"
)

type OutputParseFn func(r *bufio.Reader) (out interface{}, err error)

const exitStatusUndefined = -1

// Run is a simple method to run external program and don't care about its output or exit status
func Run(ctx context.Context, logFn utils.LogFnType, client *ssh.Client, cmd string, env map[string]string) error {
	_, exit, err := RunAndParse(ctx, logFn, client,
		cmd, env, ParseDiscard)

	if err != nil {
		return trace.Wrap(err, cmd)
	}

	if exit != 0 {
		return trace.Errorf("%s returned %d", cmd, exit)
	}

	return nil
}

// RunAndParse runs remote SSH command with environment variables set by `env`
// exitStatus is -1 if undefined
func RunAndParse(ctx context.Context, logFn utils.LogFnType, client *ssh.Client, cmd string, env map[string]string, parse OutputParseFn) (out interface{}, exitStatus int, err error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, exitStatusUndefined, trace.Wrap(err, "ssh session")
	}
	defer session.Close()

	envStrings := []string{}
	if env != nil {
		for k, v := range env {
			envStrings = append(envStrings, fmt.Sprintf("%s=%s", k, v))
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

	outCh := make(chan interface{}, 1)
	if parse != nil {
		go func() {
			out, err := parse(bufio.NewReader(
				&readLogger{fmt.Sprintf("[%v] %s [stdout]", client.RemoteAddr(), cmd), logFn, stdout}))
			if err != nil {
				outCh <- trace.Wrap(err)
			} else {
				outCh <- out
			}
		}()
	} else {
		outCh <- nil
	}

	runCh := make(chan error, 1)
	go func() {
		logFn("[%v] (starting) %s", client.RemoteAddr(), cmd)
		runCh <- session.Run(fmt.Sprintf("%s %s", strings.Join(envStrings, " "), cmd))
	}()

	go func() {
		r := bufio.NewReader(stderr)
		for {
			line, err := r.ReadString('\n')
			logFn("[%v] %s [stderr] %s", client.RemoteAddr(), cmd, line)
			if parse == nil {
				logFn("[%v] %s [-----]: closing session", client.RemoteAddr(), cmd)
				session.Close()
				logFn("[%v] %s [-----]: closed", client.RemoteAddr(), cmd)
				runCh <- nil // FIXME : this is a hack; session closure does not unblock session.Run() wonder if there's a better way
				return
			}
			if err != nil {
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		err := session.Signal(ssh.SIGTERM)
		return nil, exitStatusUndefined,
			trace.Errorf("[%v] %s timed out, sending SIGTERM: %v",
				client.RemoteAddr(), cmd, err)
	case err = <-runCh:
		if exitErr, isExitErr := err.(*ssh.ExitError); isExitErr {
			logFn("[%v] (exit=%d) %s", client.RemoteAddr(), exitErr.ExitStatus(), cmd)
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
			logFn("[%v] (parse fn failed) %s, error %v",
				client.RemoteAddr(), cmd, outErr)
			return nil, exitStatusUndefined, trace.Wrap(outErr)
		}
		logFn("[%v] (exit=0) %s", client.RemoteAddr(), cmd)
		return out, 0, nil
	}
}

func ParseDiscard(r *bufio.Reader) (interface{}, error) {
	io.Copy(ioutil.Discard, r)
	return nil, nil
}

type readLogger struct {
	prefix string
	logFn  utils.LogFnType
	r      io.Reader
}

func (l *readLogger) Read(p []byte) (n int, err error) {
	n, err = l.r.Read(p)
	if err != nil && err != io.EOF {
		l.logFn("%s %q: %v", l.prefix, p[0:n], err)
	} else {
		l.logFn("%s %q", l.prefix, p[0:n])
	}
	return n, err
}
