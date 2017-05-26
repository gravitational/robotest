package sshutils

import (
	"bufio"
	"bytes"
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

type SshNode interface {
	Client() *ssh.Client
	Logf(format string, args ...interface{})
	String() string
}

// Run is a simple method to run external program and don't care about its output or exit status
func Run(ctx context.Context, node SshNode, cmd string, env map[string]string) error {
	_, exit, err := RunAndParse(ctx, node,
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
func RunAndParse(ctx context.Context, node SshNode, cmd string, env map[string]string, parse OutputParseFn) (out interface{}, exitStatus int, err error) {
	session, err := node.Client().NewSession()
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

	session.Stdin = new(bytes.Buffer)

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
				&readLogger{fmt.Sprintf("%s [stdout]", cmd), node.Logf, stdout}))
			if err != nil {
				outCh <- trace.Wrap(err)
			} else {
				outCh <- out
			}
		}()
	} else {
		outCh <- nil
	}

	runCh := make(chan error, 3)
	go func() {
		node.Logf("(starting) %s", cmd)
		runCh <- session.Run(fmt.Sprintf("%s %s", strings.Join(envStrings, " "), cmd))
	}()

	go func() {
		r := bufio.NewReader(stderr)
		for {
			line, err := r.ReadString('\n')
			if line != "" {
				node.Logf("%s [stderr] %s", cmd, line)
			}
			if parse == nil {
				session.Close()
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
				node, cmd, err)
	case err = <-runCh:
		if exitErr, isExitErr := err.(*ssh.ExitError); isExitErr {
			node.Logf("(exit=%d) %s", exitErr.ExitStatus(), cmd)
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
			node.Logf("(parse fn failed) %s, error %v",
				cmd, outErr)
			return nil, exitStatusUndefined, trace.Wrap(outErr)
		}
		node.Logf("(exit=0) %s", cmd)
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
	} else if n > 0 {
		l.logFn("%s %q", l.prefix, p[0:n])
	}
	return n, err
}
