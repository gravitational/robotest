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
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/configure"
	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/trace"
)

func Stop(confS string) (err error) {
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
