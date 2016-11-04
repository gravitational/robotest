package robotest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/configure"
	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/trace"
)

type config struct {
	// AccessKey is AWS access key
	AccessKey string `json:"access_key" env:"ROBO_ACCESS_KEY"`
	// SecretKey is AWS secret key
	SecretKey string `json:"secret_key" env:"ROBO_SECRET_KEY"`
	// Region is AWS region to deploy to
	Region string `json:"region" env:"ROBO_REGION"`
	// KeyPair is AWS key pair to use
	KeyPair string `json:"key_pair" env:"ROBO_KEY_PAIR"`
	// InstanceType is AWS instance type
	InstanceType string `json:"instance_type" env:"ROBO_INSTANCE_TYPE"`
	// ClusterName is the name that will be assigned to the provisioned AWS machines
	ClusterName string `json:"cluster_name" env:"ROBO_CLUSTER_NAME"`
	// Nodes is the number of nodes to provision
	Nodes int `json:"nodes" env:"ROBO_NODES"`
	// InstallerURL is AWS S3 URL with the installer
	InstallerURL string `json:"installer_url" env:"ROBO_INSTALLER_URL"`
	// FlavorLabel is the "label" of the flavor to install (important: "label", not "name")
	FlavorLabel string `json:"flavor_label" env:"ROBO_FLAVOR_LABEL"`
	// SSHKeyPath is the path to the private SSH key to use to connect to provisioned machines
	SSHKeyPath string `json:"ssh_key_path" env:"ROBO_SSH_KEY_PATH"`
	// TerraformPath is the path to the terraform script to use for provisioning
	TerraformPath string `json:"terraform_path" env:"ROBO_TERRAFORM_PATH"`
	// LicensePath is the path to the file containing license
	LicensePath string `json:"license_path" env:"ROBO_LICENSE_PATH"`
}

// processConfig validates that all required fields are set
func processConfig(conf *config) error {
	var errors []error
	if conf.AccessKey == "" {
		errors = append(errors, trace.BadParameter("access_key is missing"))
	}
	if conf.SecretKey == "" {
		errors = append(errors, trace.BadParameter("secret_key is missing"))
	}
	if conf.ClusterName == "" {
		errors = append(errors, trace.BadParameter("cluster_name is missing"))
	}
	if conf.InstallerURL == "" {
		errors = append(errors, trace.BadParameter("installer_url is missing"))
	}
	if conf.FlavorLabel == "" {
		errors = append(errors, trace.BadParameter("flavor_label is missing"))
	}
	if conf.SSHKeyPath == "" {
		errors = append(errors, trace.BadParameter("ssh_key_path is missing"))
	}
	if conf.TerraformPath == "" {
		errors = append(errors, trace.BadParameter("terraform_path is missing"))
	}
	return trace.NewAggregate(errors...)
}

func Start(confS string) (err error) {
	var conf config
	err = json.Unmarshal([]byte(confS), &conf)
	if err != nil {
		return trace.Wrap(err)
	}

	err = configure.ParseEnv(&conf)
	if err != nil {
		return trace.Wrap(err)
	}

	err = processConfig(&conf)
	if err != nil {
		return trace.Wrap(err)
	}

	dir, err := ioutil.TempDir("", fmt.Sprintf("robotest-%v-", conf.ClusterName))
	if err != nil {
		return trace.Wrap(err)
	}
	log.Infof("state dir: %v", dir)

	tf, err := provisionTerraform(dir, conf.TerraformPath, conf)

	// cleanup to destroy any infrastructure that we might fully or partially create with terraform
	defer func() {
		err := destroyTerraform(dir, conf)
		if err != nil {
			log.Errorf("failed to destroy infrastructure: %v", trace.DebugReport(err))
			return
		}
		// only in the case of successful deprovisioning should we remove state dir so that there is a way to cleanup
		os.RemoveAll(dir)
	}()

	// handle provisioning errors
	if err != nil {
		return trace.Wrap(err)
	}

	log.Infof("provisioned machines: installer = %v, private ips = %v, public ips = %v",
		tf.installerIP, tf.privateIPs, tf.publicIPs)

	var session *ssh.Session
	err = utils.Retry(time.Second, 100, func() error {
		session, err = connectSSH(tf.installerIP, conf.SSHKeyPath)
		return trace.Wrap(err)
	})
	if err != nil {
		return trace.Wrap(err)
	}
	defer session.Close()

	log.Infof("connected to the installer machine via ssh")

	stdin, err := session.StdinPipe()
	if err != nil {
		return trace.Wrap(err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return trace.Wrap(err)
	}

	var out bytes.Buffer
	go io.Copy(io.MultiWriter(os.Stdout, &out), stdout)

	stderr, err := session.StderrPipe()
	if err != nil {
		return trace.Wrap(err)
	}
	go io.Copy(os.Stderr, stderr)

	// value received at this channel will indicate test completion (success or failure)
	doneC := make(chan error)

	// this goroutine handles the installation procedure
	go func(doneC chan error) {
		// select the first interface
		for !strings.Contains(out.String(), "select interface number") {
			time.Sleep(time.Second)
		}
		io.Copy(stdin, strings.NewReader("1\n"))

		// confirm
		for !strings.Contains(out.String(), "confirm (yes/no)") {
			time.Sleep(time.Second)
		}
		io.Copy(stdin, strings.NewReader("yes\n"))

		// find installer URL
		for !strings.Contains(out.String(), "OPEN THIS IN BROWSER") {
			time.Sleep(time.Second)
		}
		match := reInstallerURL.FindStringSubmatch(out.String())
		if len(match) != 2 {
			doneC <- fmt.Errorf("failed to extract installer URL")
			return
		}

		urlS := match[1]
		u, err := url.Parse(urlS)
		if err != nil {
			doneC <- fmt.Errorf("failed to parse url %v: %v", urlS, trace.DebugReport(err))
			return
		}

		// generated installer URL has private IP in it. replace it with the public IP of the
		// installer machine so we can connect to it
		host, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			doneC <- fmt.Errorf("failed to split host:port in %v: %v", urlS, trace.DebugReport(err))
			return
		}
		host = tf.installerIP
		u.Host = strings.Join([]string{host, port}, ":")

		urlS = u.String()
		err = runSelenium(urlS, conf, *tf)
		if err != nil {
			doneC <- fmt.Errorf("test failure: %v", trace.DebugReport(err))
			return
		}

		doneC <- nil
	}(doneC)

	// launch the installer
	err = session.Start(installerCommand)
	if err != nil {
		return trace.Wrap(err)
	}

	// wait for the result
	err = <-doneC
	if err != nil {
		return trace.Wrap(err, "test finished with error")
	}

	log.Infof("great success!")
	return nil
}

// connectSSH establishes SSH connection and returns a session
func connectSSH(ip, keyPath string) (*ssh.Session, error) {
	keyBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	key, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	conf := &ssh.ClientConfig{
		User: "centos",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		Timeout: 10 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", ip), conf)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return session, nil
}

// startAgent runs the provided gravity agent command on the specified machine
func startAgent(ip, command, keyPath string) (err error) {
	defer func() {
		if err != nil {
			trace.DebugReport(err)
		}
	}()

	session, err := connectSSH(ip, keyPath)
	if err != nil {
		return trace.Wrap(err)
	}
	defer session.Close()

	err = session.Run(command)
	if err != nil {
		return trace.Wrap(err)
	}

	return nil
}

var (
	reInstallerURL = regexp.MustCompile("(?m:^OPEN THIS IN BROWSER: (.+)$)")
)

// installerCommand waits for the installer tarball to download, unpacks it and launches the installation
const installerCommand = `while [ ! -f /home/centos/installer.tar.gz ]; do sleep 5; done; \
tar -xvf /home/centos/installer.tar.gz -C /home/centos/installer; \
/home/centos/installer/install`
