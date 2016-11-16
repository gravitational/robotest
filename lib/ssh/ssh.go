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

func RunCommandWithOutput(session *ssh.Session, command string, w io.Writer) error {
	stderr, err := session.StderrPipe()
	if err != nil {
		return trace.Wrap(err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return trace.Wrap(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	sink := make(chan string)

	go func() {
		stream(stdout, sink)
		wg.Done()
	}()
	go func() {
		stream(stderr, sink)
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

	err = session.Wait()
	wg.Wait()

	return trace.Wrap(err)
}

func stream(r io.Reader, sink chan<- string) {
	reader := bufio.NewReader(r)
	var err error
	for err == nil {
		var line string
		line, err = reader.ReadString('\n')
		if err == io.EOF {
			err = nil
			break
		}
		sink <- line
	}
	if err != nil {
		log.Errorf("failed to stream: %v", err)
	}
}

const sshConnectTimeout = 20 * time.Second
