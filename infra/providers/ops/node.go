package ops

import (
	"fmt"
	"os"

	"github.com/gravitational/robotest/infra"
	sshutils "github.com/gravitational/robotest/lib/ssh"

	"github.com/gravitational/trace"
	"golang.org/x/crypto/ssh"
)

type node struct {
	publicIP   string
	privateIP  string
	sshKeyPath string
	sshUser    string
}

func New(publicIP string, privateIP string, sshUser string, sshKeyPath string) infra.Node {
	res := &node{
		publicIP:   publicIP,
		privateIP:  privateIP,
		sshKeyPath: sshKeyPath,
		sshUser:    sshUser,
	}

	return res
}

func (r *node) Addr() string {
	return r.publicIP
}

func (r *node) PrivateAddr() string {
	return r.privateIP
}

func (r *node) Connect() (*ssh.Session, error) {
	client, err := r.Client()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return client.NewSession()
}

func (r *node) Client() (*ssh.Client, error) {
	keyFile, err := os.Open(r.sshKeyPath)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return sshutils.Client(fmt.Sprintf("%v:22", r.publicIP), r.sshUser, keyFile)
}

func (r node) String() string {
	return fmt.Sprintf("node(addr=%v)", r.publicIP)
}
