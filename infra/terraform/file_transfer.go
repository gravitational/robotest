package terraform

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gravitational/trace"
)

/*
 FIXME: there are still few fundamental issues with this approach to SSH,
 and it need be replaced with command by command execution

 1. ability to kill hung remote command (i.e. ./gravity wizard)
    otherwise robotest completes but node is unsusable as there's still background process running

 2. waiting for boostrap script to complete (or maybe replacing its behind-the-scenes with more explicit preparation here)

 3. being sure which command exactly failed and avoiding monstrous && style pipelining

 4. support for SCP for local tarballs
*/

// TransferFile takes file URL which may be S3 or HTTP or local file and transfers it to the machine
// fileUrl - file to download, could be S3:// or http(s)://
// command - to run out of installer package
func (t *terraform) makeRemoteCommand(fileUrl, command string) (string, error) {
	u, err := url.Parse(fileUrl)
	if err != nil {
		return "", trace.Wrap(err, "Parsing %s", fileUrl)
	}

	var fetchCmd string
	var homeDir = fmt.Sprintf("/home/%s", t.sshUser)
	var outFile string
	if strings.HasSuffix(fileUrl, ".tar.gz") {
		outFile = fmt.Sprintf("%s/installer.tar.gz", homeDir)
	} else if strings.HasSuffix(fileUrl, ".tar") {
		outFile = fmt.Sprintf("%s/installer.tar", homeDir)
	} else {
		return "", trace.Errorf("Unsupported installer packaging %s", fileUrl)
	}

	switch u.Scheme {
	case "s3":
		if t.Config.AWS == nil {
			return "", trace.Errorf("AWS config missing, cannot use S3 URLs %s", fileUrl)
		}

		fetchCmd = fmt.Sprintf(`AWS_ACCESS_KEY_ID=%s \
			AWS_SECRET_ACCESS_KEY=%s \
			AWS_DEFAULT_REGION=%s \
			aws s3 cp %s - > %s`,
			t.Config.AWS.AccessKey, t.Config.AWS.SecretKey, t.Config.AWS.Region,
			fileUrl, outFile)
	case "http":
	case "https":
		fetchCmd = fmt.Sprintf("wget %s -O %s/", fileUrl, outFile)
	case "":
	case "gs":
	default:
		// TODO : implement SCP and GCLOUD methods
		return "", fmt.Errorf("Unsupported URL schema %s", fileUrl)
	}

	cmd := fmt.Sprintf(`echo Testing if bootstrap completed && test -f /var/lib/bootstrap_complete && \
		echo Cleaning up && rm -rf %[1]s/installer/* 
		echo Downloading installer %[5]s ... && %[2]s && \
		echo Creating installer dir && mkdir -p %[1]s/installer && \
		echo Unpacking installer && tar -xvf %[3]s -C %[1]s/installer && \
		echo Launching command %[4]s && cd %[1]s/installer && %[4]s`, homeDir, fetchCmd, outFile, command, fileUrl)

	// cmd = fmt.Sprintf(`echo Launching command %[4]s && cd %[1]s/installer && %[4]s`,
	//	homeDir, fetchCmd, outFile, command, fileUrl)

	return cmd, nil
}
