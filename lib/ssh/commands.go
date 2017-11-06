package sshutils

import (
	"context"
	"fmt"

	"github.com/gravitational/trace"

	"github.com/sirupsen/logrus"
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
func RunCommands(ctx context.Context, client *ssh.Client, log logrus.FieldLogger, commands []Cmd) error {
	for _, cmd := range commands {
		exit, err := RunAndParse(ctx, client, log, cmd.Command, cmd.Env, ParseDiscard)
		if err != nil {
			return trace.Wrap(err, cmd.Command)
		}
		if exit != 0 {
			return trace.Errorf("%s returned %d", cmd.Command, exit)
		}
	}
	return nil
}

const (
	tmpDir = "/tmp"
	SUDO   = true
)

// RunScript will run a .sh script on remote host
// if script should not be executed it should have internal flag files and terminate
func RunScript(ctx context.Context, client *ssh.Client, log logrus.FieldLogger, scriptPath string, sudo bool) error {
	remotePath, err := PutFile(ctx, client, log, scriptPath, tmpDir)
	if err != nil {
		return trace.Wrap(err)
	}

	cmd := fmt.Sprintf("/bin/bash -x %s", remotePath)
	if sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}

	err = Run(ctx, client, log, cmd, nil)
	return trace.Wrap(err)
}
