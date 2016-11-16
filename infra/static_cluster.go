package infra

import (
	"io"
	"strings"

	"golang.org/x/crypto/ssh"

	sshutils "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/trace"
)

type staticCluster struct {
	nodes        []string
	user         string
	key          string
	opsCenterURL string
}

func (r *staticCluster) Installer() Node      { return nil }
func (r *staticCluster) NumNodes() int        { return len(r.nodes) }
func (r *staticCluster) OpsCenterURL() string { return r.opsCenterURL }

func (r *staticCluster) Nodes() []Node {
	return nil
}

func (r *staticCluster) Run(command string) error {
	// TODO
	// return RunOnNodes(command, r.nodes)
	return nil
}

func (r *staticCluster) Close() error {
	return nil
}

func newStaticNode(addr string, cluster *staticCluster) Node {
	return &staticNode{addr: addr, staticCluster: cluster}
}

type staticNode struct {
	*staticCluster

	addr string
}

func (r *staticNode) Connect() (*ssh.Session, error) {
	return sshutils.Connect(r.addr, r.user, strings.NewReader(r.key))
}

func (r *staticNode) Run(command string, w io.Writer) error {
	session, err := sshutils.Connect(r.addr, r.user, strings.NewReader(r.key))
	if err != nil {
		return trace.Wrap(err)
	}
	defer session.Close()

	return sshutils.RunCommandWithOutput(session, command, w)
}
