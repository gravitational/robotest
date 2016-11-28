package system

import (
	"io"
	"os/exec"
	"strings"

	"github.com/gravitational/robotest/lib/constants"
	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
)

type CommandOptionSetter func(cmd *exec.Cmd)

func Dir(dir string) CommandOptionSetter {
	return func(cmd *exec.Cmd) {
		cmd.Dir = dir
	}
}

func ExecL(cmd *exec.Cmd, out io.Writer, entry *log.Entry, setters ...CommandOptionSetter) error {
	err := Exec(cmd, out, setters...)
	entry.WithFields(log.Fields{
		constants.FieldCommandError:       (err != nil),
		constants.FieldCommandErrorReport: trace.UserMessage(err),
	}).Info(strings.Join(cmd.Args, " "))
	return err
}

func Exec(cmd *exec.Cmd, out io.Writer, setters ...CommandOptionSetter) error {
	return ExecWithInput(cmd, "", out, setters...)
}

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
		io.WriteString(stdin, input)
	}

	if err := cmd.Wait(); err != nil {
		return trace.Wrap(err)
	}

	return nil
}
