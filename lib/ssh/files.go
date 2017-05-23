package sshutils

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	"github.com/gravitational/robotest/lib/utils"

	"github.com/gravitational/trace"

	"golang.org/x/crypto/ssh"
)

// TransferFile takes file URL which may be S3 or HTTP or local file and transfers it to remote the machine
// fileUrl - file to download, could be S3:// or http(s)://
func TransferFile(ctx context.Context, logFn utils.LogFnType, client *ssh.Client, fileUrl, dstDir string, env map[string]string) (path string, err error) {
	u, err := url.Parse(fileUrl)
	if err != nil {
		return "", trace.Wrap(err, "parsing %s", fileUrl)
	}

	fname := filepath.Base(u.Path)
	dstPath := filepath.Join(dstDir, fname)
	var cmd string
	switch u.Scheme {
	case "s3":
		cmd = fmt.Sprintf(`aws s3 cp %s - > %s`, fileUrl, dstPath)
	case "http":
	case "https":
		cmd = fmt.Sprintf("wget %s -O %s/", fileUrl, dstPath)
	case "":
		remotePath, err := PutFile(ctx, logFn, client, fileUrl, dstDir)
		return remotePath, trace.Wrap(err)
	case "gs":
	default:
		// TODO : implement SCP and GCLOUD methods
		return "", fmt.Errorf("unsupported URL schema %s", fileUrl)
	}

	err = RunCommands(ctx, logFn, client, []Cmd{
		{fmt.Sprintf("mkdir -p %s", dstDir), nil},
		{cmd, env},
	})
	if err == nil {
		return dstPath, nil
	}
	return "", trace.Wrap(err)
}

const (
	// TestRegularFile file exists and is a regular file
	TestRegularFile = "-f"
	// TestDir file exists and is a directory
	TestDir = "-d"
)

// TestFile tests remote file using `test` command.
// It returns trace.NotFound in case test fails, nil is test passes, and unspecified error otherwise
func TestFile(ctx context.Context, logFn utils.LogFnType, client *ssh.Client, path, test string) error {
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
		return trace.Errorf("[%v] %s returned exit code %d", client.RemoteAddr(), cmd, exit)
	}
}

// WaitForFile waits for a test to become true against a remote file (or context to expire)
func WaitForFile(ctx context.Context, logFn utils.LogFnType, client *ssh.Client, path, test string, sleepDuration time.Duration) error {
	for {
		err := TestFile(ctx, logFn,
			client, path, test)

		if trace.IsNotFound(err) {
			logFn("[%v] waiting for %s, will retry in %v", client.RemoteAddr(), path, sleepDuration)
			select {
			case <-ctx.Done():
				return trace.Errorf("[%v] timed out waiting for %s", client.RemoteAddr(), path)
			case <-time.After(sleepDuration):
				continue
			}
		}

		if err == nil {
			return nil
		}

		return trace.Wrap(err, "[%v] waiting for %s", client.RemoteAddr(), path)
	}
}
