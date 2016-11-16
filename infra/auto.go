package infra

import (
	"strings"

	"golang.org/x/crypto/ssh"

	sshutils "github.com/gravitational/robotest/lib/ssh"
)

// autoCluster represents a cluster managed by an active OpsCenter
type autoCluster struct {
	nodes        []string
	user         string
	key          string
	opsCenterURL string
}

func (r *autoCluster) Installer() Node      { return nil }
func (r *autoCluster) OpsCenterURL() string { return r.opsCenterURL }

func (r *autoCluster) Provisioner() Provisioner {
	// TODO: implement
	return nil
}

func (r *autoCluster) Close() error {
	return nil
}

func newAutoNode(addr string, cluster *autoCluster) Node {
	return &autoNode{addr: addr, autoCluster: cluster}
}

type autoNode struct {
	*autoCluster

	addr string
}

func (r *autoNode) Connect() (*ssh.Session, error) {
	return sshutils.Connect(r.addr, r.user, strings.NewReader(r.key))
}
