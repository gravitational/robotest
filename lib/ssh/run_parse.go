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

type OutputParseFn func(r *bufio.Reader) error

const exitStatusUndefined = -1

type SshNode interface {
	Client() *ssh.Client
	Logf(format string, args ...interface{})
	String() string
}

// Run is a simple method to run external program and don't care about its output or exit status
func Run(ctx context.Context, node SshNode, cmd string, env map[string]string) error {
	exit, err := RunAndParse(ctx, node,
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
func RunAndParse(ctx context.Context, node SshNode, cmd string, env map[string]string, parse OutputParseFn) (exitStatus int, err error) {
	session, err := node.Client().NewSession()
	if err != nil {
		return exitStatusUndefined, trace.Wrap(err, "ssh session")
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
		return exitStatusUndefined, trace.Wrap(err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return exitStatusUndefined, trace.Wrap(err)
	}

	errCh := make(chan error, 2)
	expectErrs := 1
	if parse != nil {
		expectErrs++
		go func() {
			err := parse(bufio.NewReader(
				&readLogger{fmt.Sprintf("%s [stdout]", cmd), node.Logf, stdout}))
			errCh <- trace.Wrap(err)
		}()
	}

	go func() {
		node.Logf("(starting) %s", cmd)
		errCh <- session.Run(fmt.Sprintf("%s %s", strings.Join(envStrings, " "), cmd))
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
		case ctxErr := <-ctx.Done():
			err := session.Signal(ssh.SIGTERM)
			err = trace.Errorf("[%v] %s %v, sending SIGTERM: %v",
				node, cmd, ctxErr, err)
			node.Logf(err.Error())
			return exitStatusUndefined, err
		case err = <-errCh:
			if exitErr, isExitErr := err.(*ssh.ExitError); isExitErr {
				err := trace.Errorf("(exit=%d) %s", exitErr.ExitStatus(), cmd)
				node.Logf(err.Error())
				return exitErr.ExitStatus(), err
			}
			if err != nil {
				err := trace.Wrap(err, cmd)
				node.Logf(err.Error())
				return exitStatusUndefined, err
			}
		}
	}
	node.Logf("(exit=0) %s", cmd)
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
	prefix string
	logFn  utils.LogFnType
	r      io.Reader
}

func (l *readLogger) Read(p []byte) (n int, err error) {
	n, err = l.r.Read(p)
	if err != nil && err != io.EOF {
		l.logFn("%s %s: %v", l.prefix, string(p[0:n]), err)
	} else if n > 0 {
		l.logFn("%s %s", l.prefix, string(p[0:n]))
	}
	return n, err
}
