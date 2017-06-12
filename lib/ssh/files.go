package sshutils

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	"github.com/gravitational/trace"
)

// TransferFile takes file URL which may be S3 or HTTP or local file and transfers it to remote the machine
// fileUrl - file to download, could be S3:// or http(s)://
func TransferFile(ctx context.Context, node SshNode, fileUrl, dstDir string, env map[string]string) (path string, err error) {
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
		remotePath, err := PutFile(ctx, node, fileUrl, dstDir)
		return remotePath, trace.Wrap(err)
	case "gs":
	default:
		// TODO : implement SCP and GCLOUD methods
		return "", fmt.Errorf("unsupported URL schema %s", fileUrl)
	}

	err = RunCommands(ctx, node, []Cmd{
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
func TestFile(ctx context.Context, node SshNode, path, test string) error {
	cmd := fmt.Sprintf("sudo test %s %s", test, path)
	_, exit, err := RunAndParse(ctx, node, cmd, nil, ParseDiscard)

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
	}

	return trace.Wrap(err, cmd)
}

// WaitForFile waits for a test to become true against a remote file (or context to expire)
func WaitForFile(ctx context.Context, node SshNode, path, test string, sleepDuration time.Duration) error {
	for {
		err := TestFile(ctx, node, path, test)

		if trace.IsNotFound(err) {
			node.Logf("waiting for %s, will retry in %v", path, sleepDuration)
			select {
			case <-ctx.Done():
				return trace.Errorf("[%v] timed out waiting for %s", node, path)
			case <-time.After(sleepDuration):
				continue
			}
		}

		if err == nil {
			return nil
		}

		return trace.Wrap(err, "[%v] waiting for %s", node, path)
	}
}
