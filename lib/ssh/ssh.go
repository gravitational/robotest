package sshutils

import (
	"bufio"
	"io"
	"io/ioutil"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
)

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
		Timeout: sshConnectTimeout,
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
		if err != nil {
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
	sink := make(chan string)

	go func() {
		err := stream(stdout, sink)
		if err != nil {
			log.Error(err.Error())
		}
		wg.Done()
	}()
	go func() {
		err := stream(stderr, sink)
		if err != nil {
			log.Error(err.Error())
		}
		wg.Done()
	}()
	go func() {
		w := bufio.NewWriter(w)
		for line := range sink {
			_, err := w.WriteString(line)
			if err != nil {
				log.Warningf("failed to write to w: %v", err)
			}
		}
		w.Flush()
	}()

	err = session.Start(command)
	if err != nil {
		return trace.Wrap(err, "failed to start %q", command)
	}

	err = session.Wait()
	session.Close()
	wg.Wait()
	close(sink)

	return trace.Wrap(err)
}

func stream(r io.Reader, sink chan<- string) error {
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := s.Text()
		sink <- line
	}
	if err := s.Err(); err != nil && err != io.EOF {
		return trace.Wrap(err, "failed to stream")
	}
	return nil
}

const sshConnectTimeout = 20 * time.Second
