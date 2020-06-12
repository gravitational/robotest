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
	"bufio"
	"io"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"time"

	"github.com/gravitational/robotest/lib/defaults"

	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// Client creates a new SSH client specified by
// addr and user. keyInput defines the SSH key to use for authentication.
// Returns a SSH client
func Client(addr, user string, signer ssh.Signer) (*ssh.Client, error) {
	return client(addr, user, signer, realTimeoutDialer)
}

// Connect connects to remote SSH server and returns new session
func Connect(addr, user string, signer ssh.Signer) (*ssh.Session, error) {
	client, err := Client(addr, user, signer)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return session, nil
}

// RunCommandWithOutput executes the specified command in given session and
// streams session's Stderr/Stdout into w.
// The function takes ownership of session and will destroy it upon completion of
// the command
func RunCommandWithOutput(session *ssh.Session, log logrus.FieldLogger, command string, w io.Writer) (err error) {
	defer func() {
		if err != nil && session != nil {
			errClose := session.Close()
			if errClose != nil {
				log.WithError(err).Error("failed to close SSH session")
			}
		}
	}()
	var stderr io.Reader
	stderr, err = session.StderrPipe()
	if err != nil {
		return trace.Wrap(err)
	}

	var stdout io.Reader
	stdout, err = session.StdoutPipe()
	if err != nil {
		return trace.Wrap(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	errCh := make(chan error, 2)
	sink := make(chan string)
	done := make(chan struct{})

	go func() {
		errCh <- stream(stdout, sink)
		wg.Done()
	}()
	go func() {
		errCh <- stream(stderr, sink)
		wg.Done()
	}()
	go func() {
		wg.Wait()
		close(errCh)
	}()
	go func() {
		w := bufio.NewWriter(w)
		for line := range sink {
			_, err := w.Write([]byte(line))
			if err != nil {
				log.Errorf("failed to write to w: %v", err)
			}
		}
		w.Flush()
		close(done)
	}()

	err = session.Start(command)
	if err != nil {
		return trace.Wrap(err, "failed to start %q", command)
	}

	err = session.Wait()
	session.Close()
	session = nil // Avoid second close
	for err := range errCh {
		if err != nil {
			log.Errorf("failed to stream: %v", err)
		}
	}
	close(sink)
	<-done

	return trace.Wrap(err)
}

// MakePrivateKeySignerFromFile creates a singer from the specified path
func MakePrivateKeySignerFromFile(path string) (ssh.Signer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer f.Close()

	return MakePrivateKeySignerFromReader(f)
}

// MakePrivateKeySignerFromReader creates a singer from the specified reader
func MakePrivateKeySignerFromReader(r io.Reader) (ssh.Signer, error) {
	keyBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	key, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return key, nil
}

func stream(r io.Reader, sink chan<- string) error {
	s := bufio.NewScanner(r)
	s.Split(bytesID)
	for s.Scan() {
		line := s.Bytes()
		// Copy to avoid re-using scanner's internal buffer
		sink <- string(line)
	}
	if err := s.Err(); err != nil && err != io.EOF {
		return trace.Wrap(err, "failed to stream")
	}
	return nil
}

func bytesID(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Request more data
	return len(data), data, nil
}

func client(addr, user string, signer ssh.Signer, dialer sshDialer) (*ssh.Client, error) {
	conf := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		Timeout: defaults.SSHConnectTimeout,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	return dialer.Dial("tcp", addr, conf)
}

const (
	defaultDialTimeout = 30 * time.Second
	defaultKeepAlive   = 5 * time.Second
)

// Interface to allow mocking of ssh.Dial, for testing SSH
type sshDialer interface {
	Dial(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error)
}

// Real implementation of sshDialer
type realSSHDialer struct{}

var _ sshDialer = &realSSHDialer{}

func (r *realSSHDialer) Dial(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	d := &net.Dialer{
		Timeout:   config.Timeout,
		KeepAlive: defaultKeepAlive,
	}
	conn, err := d.Dial(network, addr)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(defaultDialTimeout)); err != nil {
		logrus.WithFields(logrus.Fields{
			"network": network,
			"addr":    addr,
		}).Warnf("Failed to set read deadline: %v.", err)
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	_ = conn.SetReadDeadline(time.Time{})
	return ssh.NewClient(c, chans, reqs), nil
}

// timeoutDialer wraps an sshDialer with a timeout around Dial(). The golang
// ssh library can hang indefinitely inside the Dial() call (see issue #23835).
// Wrapping all Dial() calls with a conservative timeout provides safety against
// getting stuck on that.
type timeoutDialer struct {
	dialer  sshDialer
	timeout time.Duration
}

// 150 seconds is longer than the underlying default TCP backoff delay (127
// seconds). This timeout is only intended to catch otherwise uncaught hangs.
// See: https://github.com/kubernetes/kubernetes/issues/23835 for details.
const sshDialTimeout = 150 * time.Second

var realTimeoutDialer sshDialer = &timeoutDialer{&realSSHDialer{}, sshDialTimeout}

func (r *timeoutDialer) Dial(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	config.Timeout = r.timeout
	client, err := r.dialer.Dial(network, addr, config)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return client, nil
}
