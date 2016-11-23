package sshutils

import (
	"bufio"
	"io"
	"io/ioutil"
	"sync"

	"golang.org/x/crypto/ssh"

	"github.com/gravitational/robotest/lib/defaults"
	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
)

// Connect creates a new connection to the remote system specified with
// addr and user. keyInput defines the SSH key to use for authentication.
// Returns a new session object if successful.
func Connect(addr, user string, keyInput io.Reader) (*ssh.Session, error) {
	keyBytes, err := ioutil.ReadAll(keyInput)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	key, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	conf := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		Timeout: defaults.SSHConnectTimeout,
	}

	client, err := ssh.Dial("tcp", addr, conf)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	log.Infof("connected to %v@%v", user, addr)
	return session, nil
}

// RunCommandWithOutput executes the specified command in given session and
// streams session's Stderr/Stdout into w.
// The function takes ownership of session and will destroy it upon completion of
// the command
func RunCommandWithOutput(session *ssh.Session, command string, w io.Writer) (err error) {
	defer func() {
		if err != nil && session != nil {
			errClose := session.Close()
			if errClose != nil {
				log.Errorf("failed to close SSH session: %v", errClose)
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
