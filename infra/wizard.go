package infra

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/gravitational/robotest/lib/defaults"
	"github.com/gravitational/robotest/lib/wait"
	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
)

func startWizard(provisioner Provisioner) (output *ProvisionerOutput, err error) {
	output, err = provisioner.Create()

	// destroy (partially) created infrastructure
	defer func() {
		if err == nil {
			return
		}

		errDestroy := provisioner.Destroy()
		if errDestroy != nil {
			log.Errorf("failed to destroy infrastructure: %v", trace.DebugReport(errDestroy))
		}
		if err == nil {
			err = trace.Wrap(errDestroy)
		}
	}()

	// handle provisioning errors
	if err != nil {
		return nil, trace.Wrap(err)
	}

	log.Infof("provisioned machines: %s", output)

	var session *ssh.Session
	err = wait.Retry(defaults.RetryDelay, defaults.RetryAttempts, func() error {
		session, err = provisioner.Connect(output.InstallerIP)
		if err != nil {
			log.Warning(trace.DebugReport(err))
		}
		return trace.Wrap(err)
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer session.Close()

	var stdin io.WriteCloser
	stdin, err = session.StdinPipe()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	var stdout io.Reader
	stdout, err = session.StdoutPipe()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	reader, writer := io.Pipe()
	go func() {
		_, err := io.Copy(io.MultiWriter(os.Stdout, writer), stdout)
		if err != nil {
			log.Errorf("failed to read from remote stdout: %v", err)
		}
	}()
	defer func() {
		reader.Close()
		writer.Close()
	}()

	var stderr io.Reader
	stderr, err = session.StderrPipe()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	go func() {
		_, err := io.Copy(os.Stderr, stderr)
		if err != nil {
			log.Errorf("failed to read from remote stderr: %v", err)
		}
	}()

	// launch the installer
	err = provisioner.StartInstall(session)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	var url *url.URL
	url, err = configureWizard(reader, stdin, provisioner, *output)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	output.InstallerURL = *url

	return output, nil
}

func configureWizard(stdout io.Reader, stdin io.Writer, provisioner Provisioner, output ProvisionerOutput) (installerURL *url.URL, err error) {
	s := bufio.NewScanner(stdout)
	var state scannerState = emptyState
	var addrs []string
L:
	for s.Scan() {
		line := s.Text()
		switch state {
		case readingInterfacesState:
			if strings.HasPrefix(line, "---") {
				continue
			}
			match := reInstallerIP.FindStringSubmatch(line)
			if len(match) == 3 {
				addrs = append(addrs, match[2])
			} else {
				state = emptyState
			}
		default:
			switch {
			case strings.HasPrefix(line, "[interfaces]"):
				state = readingInterfacesState
			case strings.HasPrefix(line, "select interface number"):
				if len(addrs) == 0 {
					return nil, trace.NotFound("no network interfaces reported by the installer")
				}
				index, err := provisioner.SelectInterface(output, addrs)
				if err != nil {
					return nil, trace.Wrap(err)
				}
				_, err = io.Copy(stdin, strings.NewReader(fmt.Sprintf("%v\n", index+1)))
				if err != nil {
					return nil, trace.Wrap(err, "failed to select network interface")
				}
			case strings.HasPrefix(line, "confirm (yes/no)"):
				_, err = io.Copy(stdin, strings.NewReader("yes\n"))
				if err != nil {
					return nil, trace.Wrap(err, "failed to confirm network interface")
				}
			case strings.HasPrefix(line, "OPEN THIS IN BROWSER"):
				installerURL, err = extractInstallerURL(line, output.InstallerIP)
				if err != nil {
					return nil, trace.Wrap(err)
				}
				break L
			default:
				state = emptyState
			}
		}
	}
	return installerURL, nil
}

func extractInstallerURL(input, installerIP string) (installerURL *url.URL, err error) {
	match := reInstallerURL.FindStringSubmatch(input)
	if len(match) != 2 {
		return nil, trace.NotFound("failed to extract installer URL")
	}

	addr := match[1]
	addrURL, err := url.Parse(addr)
	if err != nil {
		return nil, trace.Wrap(err, "failed to parse URL %q", addr)
	}
	log.Infof("found installer URL: %v", addrURL.Path)

	// generated installer URL has private IP in it. replace it with the public IP of the
	// installer machine so we can connect to it
	_, port, err := net.SplitHostPort(addrURL.Host)
	if err != nil {
		return nil, trace.Wrap(err, "failed to split host:port in %q", addrURL.Host)
	}
	addrURL.Host = strings.Join([]string{installerIP, port}, ":")

	return addrURL, nil
}

type scannerState byte

const (
	emptyState             = 0
	readingInterfacesState = iota
)

var (
	reInstallerURL = regexp.MustCompile("(?m:^OPEN THIS IN BROWSER: (.+)$)")
	reInstallerIP  = regexp.MustCompile(`(\d+).\s+(\d+.\d+.\d+.\d+)`)
)

// wizardCluster implements Infra
type wizardCluster struct {
	ProvisionerOutput
	provisioner Provisioner
}

func (r *wizardCluster) Close() error {
	// FIXME: is this a Destroy?
	return r.provisioner.Destroy()
}

func (r *wizardCluster) NumNodes() int {
	// TODO: provisioner-specific?
	return len(r.ProvisionerOutput.PublicIPs)
}

func (r *wizardCluster) Nodes() []Node {
	// FIXME
	return nil
}

func (r *wizardCluster) Run(command string) error {
	return RunOnNodes(command, r.provisioner.Nodes())
}

func (r *wizardCluster) OpsCenterURL() string {
	return fmt.Sprintf("%v://%v", r.InstallerURL.Scheme, r.InstallerURL.Host)
}
