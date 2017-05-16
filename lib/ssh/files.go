package sshutils

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/gravitational/trace"

	"golang.org/x/crypto/ssh"
)

// TransferFile takes file URL which may be S3 or HTTP or local file and transfers it to remote the machine
// fileUrl - file to download, could be S3:// or http(s)://
func TransferFile(ctx context.Context, logFn LogFnType, client *ssh.Client, fileUrl, dst string, env map[string]string) error {
	u, err := url.Parse(fileUrl)
	if err != nil {
		return trace.Wrap(err, "parsing %s", fileUrl)
	}

	var cmd string
	switch u.Scheme {
	case "s3":
		cmd = fmt.Sprintf(`aws s3 cp %s - > %s`, fileUrl, dst)
	case "http":
	case "https":
		cmd = fmt.Sprintf("wget %s -O %s/", fileUrl, dst)
	case "":
	case "gs":
	default:
		// TODO : implement SCP and GCLOUD methods
		return fmt.Errorf("unsupported URL schema %s", fileUrl)
	}

	baseDir := filepath.Dir(dst)

	err = RunCommands(ctx, logFn, client, []Cmd{
		{fmt.Sprintf("mkdir -p %s", baseDir), nil},
		{cmd, env},
	})
	return trace.Wrap(err)
}

const (
	// TestRegularFile file exists and is a regular file
	TestRegularFile = "-f"
	// TestDir file exists and is a directory
	TestDir = "-d"
)

// TestFile tests remote file using `test` command.
// It returns trace.NotFound in case test fails, nil is test passes, and unspecified error otherwise
func TestFile(ctx context.Context, logFn LogFnType, client *ssh.Client, path, test string) error {
	cmd := fmt.Sprintf("test %s %s", test, path)
	_, exit, err := RunAndParse(ctx, logFn, client, cmd, nil, ParseDiscard)
	if err != nil {
		return trace.Wrap(err, cmd)
	}

	/*
	   The test utility exits with one of the following values:
	   0       expression evaluated to true.
	   1       expression evaluated to false or expression was missing.
	   >1      An error occurred.
	*/
	switch exit {
	case 0:
		return nil
	case 1:
		return trace.NotFound(path)
	default:
		return trace.Errorf("%s returned exit code %d", cmd, exit)
	}
}
