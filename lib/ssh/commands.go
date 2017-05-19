package sshutils

import (
	"context"

	"github.com/gravitational/trace"

	"golang.org/x/crypto/ssh"
)

// Cmd defines command to execute on remote host
type Cmd struct {
	// Command is what is passed to remote shell
	Command string
	// Env are any environment variables to be set
	// Note that environment vars must be explicitly enabled in /etc/ssh/sshd.conf `AcceptEnv` directive
	Env map[string]string
}

// RunCommands executes commands sequentially
func RunCommands(ctx context.Context, logFn LogFnType, client *ssh.Client, commands []Cmd) error {
	for _, cmd := range commands {
		_, exit, err := RunAndParse(ctx, logFn, client, cmd.Command, cmd.Env, ParseDiscard)
		if err != nil {
			return trace.Wrap(err, cmd.Command)
		}
		if exit != 0 {
			return trace.Errorf("[%v] %s returned %d", client.RemoteAddr(), cmd.Command, exit)
		}
	}
	return nil
}
