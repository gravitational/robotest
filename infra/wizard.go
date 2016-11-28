package infra

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/gravitational/robotest/lib/defaults"
	"github.com/gravitational/robotest/lib/loc"
	"github.com/gravitational/robotest/lib/wait"
	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
)

func startWizard(provisioner Provisioner, installer Node) (cluster *wizardCluster, err error) {
	var session *ssh.Session
	err = wait.Retry(defaults.RetryDelay, defaults.RetryAttempts, func() error {
		session, err = installer.Connect()
		if err != nil {
			log.Warning(trace.DebugReport(err))
		}
		return trace.Wrap(err)
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer func() {
		if err == nil {
			return
		}
		errClose := session.Close()
		if errClose != nil {
			log.Errorf("failed to close wizard SSH session: %v", errClose)
		}
	}()

	var stdin io.WriteCloser
	stdin, err = session.StdinPipe()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer stdin.Close()

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
		reader.Close()
		writer.Close()
	}()
	defer func() {
		if err != nil {
			reader.Close()
			writer.Close()
		}
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

	var installerURL *url.URL
	installerURL, err = configureWizard(reader, stdin, provisioner, installer)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	var application *loc.Locator
	application, err = extractPackage(*installerURL)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	// Discard all stdout content after the necessary wizard details have been obtained
	go func() {
		io.Copy(ioutil.Discard, reader)
	}()

	// TODO: make sure that all io.Copy goroutines shutdown in Close
	return &wizardCluster{
		provisioner:  provisioner,
		application:  *application,
		installerURL: *installerURL,
		session:      session,
	}, nil
}

func configureWizard(stdout io.Reader, stdin io.Writer, provisioner Provisioner, installerNode Node) (installerURL *url.URL, err error) {
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
				index, err := provisioner.SelectInterface(installerNode, addrs)
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
				installerURL, err = extractInstallerURL(line, installerNode.Addr())
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

func extractPackage(installerURL url.URL) (application *loc.Locator, err error) {
	packageSuffix := strings.TrimPrefix(installerURL.Path, "/web/installer/new/")
	fields := strings.Split(packageSuffix, "/")
	if len(fields) != 3 {
		return nil, trace.Wrap(err, "invalid application package suffix %q, expected repository/name/version",
			packageSuffix)
	}
	repository, name, version := fields[0], fields[1], fields[2]

	return loc.NewLocator(repository, name, version), nil
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

	// generated installer URL has private IP in it - replace it with the public IP of the
	// installer machine to be able to connect
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

func (r *wizardCluster) Close() error {
	return r.session.Close()
}

func (r *wizardCluster) Destroy() error {
	return r.provisioner.Destroy()
}

func (r *wizardCluster) OpsCenterURL() string {
	url := r.installerURL
	url.RawQuery = ""
	url.Path = ""
	return url.String()
}

func (r *wizardCluster) Provisioner() Provisioner {
	return r.provisioner
}

func (r *wizardCluster) Config() Config {
	return r.config
}

// wizardCluster implements Infra
type wizardCluster struct {
	config       Config
	provisioner  Provisioner
	session      *ssh.Session
	installerURL url.URL
	application  loc.Locator
}

var (
	reInstallerURL = regexp.MustCompile("(?m:^OPEN THIS IN BROWSER: (.+)$)")
	reInstallerIP  = regexp.MustCompile(`(\d+).\s+(\d+.\d+.\d+.\d+)`)
)
