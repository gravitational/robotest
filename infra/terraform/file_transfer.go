package terraform

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"text/template"

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

	var buf bytes.Buffer
	err = remoteCommandTemplate.Execute(&buf, remoteCmd{
		HomeDir:             homeDir,
		FetchCommand:        fetchCmd,
		OutputFile:          outFile,
		Command:             command,
		FileURL:             fileUrl,
		PostInstallerScript: t.Config.PostInstallerScript,
	})

	if err != nil {
		return "", trace.Wrap(err, buf.String())
	}
	return buf.String(), nil
}

var remoteCommandTemplate = template.Must(
	template.New("remote_command").Parse(`echo Testing if bootstrap completed && \
		for i in {{"{"}}1..100{{"}"}} ; \
			do test -f /var/lib/bootstrap_complete && break || \
			echo Waiting for bootstrap to complete && sleep 15 ; \
		done &&  \
		echo Cleaning up && sudo rm -rf {{.HomeDir}}/installer/* && \
        if [ ! -f {{.OutputFile}} ]; then echo Downloading installer {{.FileURL}} to {{.OutputFile}} ... && {{.FetchCommand}}; fi && \
		echo Creating installer dir && mkdir -p {{.HomeDir}}/installer && \
		echo Unpacking installer && tar -xvf {{.OutputFile}} -C {{.HomeDir}}/installer && \
		echo Checking existence of post-downloading installer script and executing it && \
		if [[ -f {{.PostInstallerScript}} ]]; then sudo bash -x {{.PostInstallerScript}}; fi && \
		echo Launching command {{.Command}} && cd {{.HomeDir}}/installer && sudo {{.Command}}`))

// remoteCmd specifies configuration for the command that is executed
// on the installer node
type remoteCmd struct {
	HomeDir             string
	FetchCommand        string
	OutputFile          string
	Command             string
	FileURL             string
	PostInstallerScript string
}
