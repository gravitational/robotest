package infra

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"sync"

	"golang.org/x/crypto/ssh"

	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/trace"
)

// New creates a new cluster from the specified config and an optional
// provisioner.
// With no provisioner, existing cluster is assumed (config.InitialCluster
// must be provided).
// Provisioner should not have its Create method called - this is done
// automatically
func New(config Config, provisioner Provisioner) (Infra, error) {
	_, err := provisioner.Create()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return &autoCluster{
		config:      config,
		provisioner: provisioner,
	}, nil
}

// NewWizard creates a new cluster using an installer tarball (which
// is assumed to be part of the configuration).
// It provisions a cluster, picks a node as installer node and starts
// a local wizard process.
// Provisioner should not have its Create method called - this is done
// automatically
func NewWizard(config Config, provisioner Provisioner) (Infra, *ProvisionerOutput, error) {
	cluster, err := startWizard(provisioner)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	cluster.config = config
	return cluster, &cluster.ProvisionerOutput, nil
}

type Provisioner interface {
	Create() (*ProvisionerOutput, error)
	Destroy() error
	Connect(addr string) (*ssh.Session, error)
	// SelectInterface returns the index (in addrs) of network address to use for
	// installation.
	// addrs is guaranteed to have at least one element
	SelectInterface(output ProvisionerOutput, addrs []string) (int, error)
	StartInstall(session *ssh.Session) error
	Nodes() []Node
	NumNodes() int
	// Allocate allocates a new node (from the pool of available nodes)
	// and returns a reference to it
	Allocate() (Node, error)
	// Deallocate places specified node back to the node pool
	Deallocate(Node) error
}

type Infra interface {
	OpsCenterURL() string
	// Close releases resources
	Close() error
	// Destroy destroys the cluster (e.g. deprovisions nodes, etc.)
	Destroy() error
	// Provisioner returns the provisioner used to manage nodes in the cluster
	Provisioner() Provisioner
	Config() Config
}

type Node interface {
	Connect() (*ssh.Session, error)
}

type ProvisionerOutput struct {
	InstallerIP  string
	PrivateIPs   []string
	PublicIPs    []string
	InstallerURL url.URL
}

func (r ProvisionerOutput) String() string {
	return fmt.Sprintf("ProvisionerOutput(installer IP=%v, private IPs=%v, public IPs=%v)",
		r.InstallerIP, r.PrivateIPs, r.PublicIPs)
}

// Distribute executes the specified command on given nodes
// and waits for execution to complete before returning
func Distribute(command string, nodes []Node) error {
	log.Infof("running %q on %v", command, nodes)
	errCh := make(chan error, len(nodes))
	wg := sync.WaitGroup{}
	wg.Add(len(nodes))
	for _, node := range nodes {
		go func(errCh chan<- error) {
			log.Infof("running on %v", node)
			errCh <- Run(node, command, os.Stderr)
			wg.Done()
		}(errCh)
	}
	wg.Wait()
	close(errCh)
	var errors []error
	for err := range errCh {
		if err != nil {
			errors = append(errors, err)
		}
	}
	return trace.NewAggregate(errors...)
}

// Run executes the specified command on node and streams
// session's Stdout/Stderr to the specified w
func Run(node Node, command string, w io.Writer) error {
	session, err := node.Connect()
	if err != nil {
		return trace.Wrap(err)
	}
	return sshutils.RunCommandWithOutput(session, command, w)
}
