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
	"net/url"
	"path/filepath"

	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// TransferFile takes file URL which may be S3 or HTTP or local file and transfers it to remote the machine
// fileUrl - file to download, could be S3:// or http(s)://
func TransferFile(ctx context.Context, client *ssh.Client, log logrus.FieldLogger, fileUrl, dstDir string, env map[string]string) (path string, err error) {
	u, err := url.Parse(fileUrl)
	if err != nil {
		return "", trace.Wrap(err, "parsing %s", fileUrl)
	}

	log = log.WithFields(logrus.Fields{"file_url": fileUrl, "dst_dir": dstDir})

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
		remotePath, err := PutFile(ctx, client, log, fileUrl, dstDir)
		return remotePath, trace.Wrap(err)
	case "gs":
	default:
		// TODO : implement SCP and GCLOUD methods
		return "", fmt.Errorf("unsupported URL schema %s", fileUrl)
	}

	err = RunCommands(ctx, client, log, []Cmd{
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
func TestFile(ctx context.Context, client *ssh.Client, log logrus.FieldLogger, path, test string) error {
	cmd := fmt.Sprintf("sudo test %s %s", test, path)
	err := RunAndParse(ctx, client, log, cmd, nil, ParseDiscard)
	if err == nil {
		// Implies exit code == 0
		return nil
	}

	if exitError, ok := trace.Unwrap(err).(ExitStatusError); ok {
		/*
		   The test utility exits with one of the following values:
		   0       expression evaluated to true.
		   1       expression evaluated to false or expression was missing.
		   >1      An error occurred.
		*/
		switch exitError.ExitStatus() {
		case 0:
			return nil
		case 1:
			return trace.NotFound(path)
		}
	}

	return trace.Wrap(err, cmd)
}

// WaitForFile waits for a test to become true against a remote file (or context to expire)
func WaitForFile(ctx context.Context, client *ssh.Client, log logrus.FieldLogger, path, test string) error {
	err := wait.Retry(ctx, func() error {
		err := TestFile(ctx, client, log, path, test)

		if err == nil {
			return nil
		}

		if trace.IsNotFound(err) {
			return wait.Continue("test %s %s false", path, test)
		}

		return wait.Abort(trace.Wrap(err, "waiting for %s", path))
	})
	return trace.Wrap(err)
}
