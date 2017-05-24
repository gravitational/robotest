package sshutils

import (
	"context"

	"github.com/gravitational/trace"
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
func RunCommands(ctx context.Context, node SshNode, commands []Cmd) error {
	for _, cmd := range commands {
		_, exit, err := RunAndParse(ctx, node, cmd.Command, cmd.Env, ParseDiscard)
		if err != nil {
			return trace.Wrap(err, cmd.Command)
		}
		if exit != 0 {
			return trace.Errorf("[%v] %s returned %d", node, cmd.Command, exit)
		}
	}
	return nil
}

const tmpDir = "/tmp"

// RunScript will run a .sh script on remote host
// if script should not be executed it should have internal flag files and terminate
func RunScript(ctx context.Context, node SshNode, scriptPath string, sudo bool) error {
	remotePath, err := PutFile(ctx, node, scriptPath, tmpDir)
	if err != nil {
		return trace.Wrap(err)
	}

	err = Run(ctx, node, remotePath, nil)
	return trace.Wrap(err)
}
