package terraform

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

type node struct {
	owner     *terraform
	publicIP  string
	privateIP string
}

func (r *node) Addr() string {
	return r.publicIP
}

func (r *node) PrivateAddr() string {
	return r.privateIP
}

func (r *node) Connect() (*ssh.Session, error) {
	return r.owner.Connect(fmt.Sprintf("%v:22", r.publicIP))
}

func (r *node) Client() (*ssh.Client, error) {
	return r.owner.Client(fmt.Sprintf("%v:22", r.publicIP))
}

func (r node) String() string {
	return fmt.Sprintf("node(addr=%v)", r.publicIP)
}
