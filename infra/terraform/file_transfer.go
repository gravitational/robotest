package terraform

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gravitational/trace"
)

// TransferFile takes file URL which may be S3 or HTTP or local file and transfers it to the machine
// fileUrl - file to download, could be S3:// or http(s)://
// command - to run out of installer package
func (t *terraform) makeRemoteCommand(fileUrl, command string) (string, error) {
	u, err := url.Parse(fileUrl)
	if err != nil {
		return "", trace.Wrap(err, "Parsing %s", fileUrl)
	}

	var homeDir = fmt.Sprintf("/home/%s", t.sshUser)
	var outFile string
	if strings.HasSuffix(fileUrl, ".tar.gz") {
		outFile = fmt.Sprintf("%s/installer.tar.gz", homeDir)
	} else if strings.HasSuffix(fileUrl, ".tar") {
		outFile = fmt.Sprintf("%s/installer.tar", homeDir)
	} else {
		return "", trace.Errorf("only .tar and .tar.gz installers supported, got %s", fileUrl)
	}

	var fetchCmd string
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
		return "", fmt.Errorf("unsupported URL schema %s", fileUrl)
	}

	// FIXME: migrate to ssh sequential command execution interface
	cmd := fmt.Sprintf(`echo Testing if bootstrap completed && \
		for i in {1..100} ; \
			do test -f /var/lib/bootstrap_complete && break || \
			echo Waiting for bootstrap to complete && sleep 15 ; \
		done &&  \
		echo Cleaning up && rm -rf %[1]s/installer/* && 
		echo Downloading installer %[5]s to %[3]s ... && %[2]s && \
		echo Creating installer dir && mkdir -p %[1]s/installer && \
		echo Unpacking installer && tar -xvf %[3]s -C %[1]s/installer && \
		echo Launching command %[4]s && cd %[1]s/installer && %[4]s`,
		homeDir, fetchCmd, outFile, command, fileUrl)

	return cmd, nil
}
