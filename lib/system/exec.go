package system

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/gravitational/robotest/lib/constants"
	"github.com/gravitational/trace"

	log "github.com/sirupsen/logrus"
)

// CommandOptionSetter defines an interface to configure child process
// before execution
type CommandOptionSetter func(cmd *exec.Cmd)

// Dir sets working directory for the child process
func Dir(dir string) CommandOptionSetter {
	return func(cmd *exec.Cmd) {
		cmd.Dir = dir
	}
}

// SetEnv passes specified environment variables to the child process
func SetEnv(envs ...string) CommandOptionSetter {
	return func(cmd *exec.Cmd) {
		if len(cmd.Env) == 0 {
			cmd.Env = os.Environ()
		}
		cmd.Env = append(cmd.Env, envs...)
	}
}

// ExecL executes the specified command and outputs its Stdout/Stderr into the specified
// writer `out`, using `logger` for logging.
// Accepts configuration as a series of CommandOptionSetters
func ExecL(cmd *exec.Cmd, out io.Writer, logger log.FieldLogger, setters ...CommandOptionSetter) error {
	err := Exec(cmd, out, setters...)
	logger.WithFields(log.Fields{
		constants.FieldCommandError:       (err != nil),
		constants.FieldCommandErrorReport: trace.UserMessage(err),
	}).Info(strings.Join(cmd.Args, " "))
	return err
}

// ExecL executes the specified command and outputs its Stdout/Stderr into the specified
// writer `out`.
// Accepts configuration as a series of CommandOptionSetters
func Exec(cmd *exec.Cmd, out io.Writer, setters ...CommandOptionSetter) error {
	return ExecWithInput(cmd, "", out, setters...)
}

// ExecWithInput executes the specified command and outputs its Stdout/Stderr into the specified
// writer `out`.
// Uses `input` to provide command with Stdin input
// Accepts configuration as a series of CommandOptionSetters
func ExecWithInput(cmd *exec.Cmd, input string, out io.Writer, setters ...CommandOptionSetter) error {
	for _, s := range setters {
		s(cmd)
	}
	execPath, err := exec.LookPath(cmd.Path)
	if err != nil {
		return trace.Wrap(err)
	}
	cmd.Path = execPath
	cmd.Stdout = out
	cmd.Stderr = out

	var stdin io.WriteCloser
	if input != "" {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			return trace.Wrap(err)
		}
		defer stdin.Close()
	}

	if err := cmd.Start(); err != nil {
		return trace.Wrap(err)
	}

	if stdin != nil {
		_, _ = io.WriteString(stdin, input)
	}

	if err := cmd.Wait(); err != nil {
		return trace.Wrap(err)
	}

	return nil
}
