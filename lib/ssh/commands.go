/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sshutils

import (
	"context"
	"fmt"

	"github.com/gravitational/robotest/lib/defaults"
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
		err := RunAndParse(ctx, client, log, cmd.Command, cmd.Env, ParseDiscard)
		if err != nil {
			return trace.Wrap(err, cmd.Command)
		}
	}
	return nil
}

const (
	SUDO = true
)

// RunScript will run a .sh script on remote host
// if script should not be executed it should have internal flag files and terminate
func RunScript(ctx context.Context, client *ssh.Client, log logrus.FieldLogger, scriptPath string, sudo bool) error {
	remotePath, err := PutFile(ctx, client, log, scriptPath, defaults.TmpDir)
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
