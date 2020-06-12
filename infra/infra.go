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

package infra

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/crypto/ssh"

	"github.com/gravitational/robotest/lib/loc"
	sshutils "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"
	log "github.com/sirupsen/logrus"
)

// New creates a new cluster from the specified config and an optional
// provisioner.
// If no provisioner is specified, automatic provisioning is assumed
func New(config Config, opsCenterURL string, provisioner Provisioner) (Infra, error) {
	return &autoCluster{
		opsCenterURL: opsCenterURL,
		provisioner:  provisioner,
	}, nil
}

// NewWizard creates a new cluster using an installer tarball (which
// is assumed to be part of the configuration).
// It provisions a cluster, picks an installer node and starts
// a local wizard process.
// Returns the reference to the created infrastructure and the application package
// the wizard is installing
func NewWizard(config Config, provisioner Provisioner, installer Node) (Infra, *loc.Locator, error) {
	cluster, err := startWizard(provisioner, installer)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	cluster.config = config
	return cluster, &cluster.application, nil
}

// Infra describes the infrastructure as used in tests.
//
// Infrastructure can be a new cluster that is provisioned as part of the test run
// using one of the built-in provisioners, or an active cluster and Ops Center
// to run tests that require existing infrastructure
type Infra interface {
	// OpsCenterURL returns the address of the Ops Center this infrastructure describes.
	// This can be an existing Ops Center or the one created using the provided provisioner
	// running the wizard
	OpsCenterURL() string
	// Close releases resources
	Close() error
	// Destroy destroys the cluster (e.g. deprovisions nodes, etc.)
	Destroy() error
	// Provisioner returns the provisioner used to manage nodes in the cluster.
	// If the provisioner is nil, the cluster is assumed to use automatic
	// provisioning
	Provisioner() Provisioner
	// Config returns a configuration this infrastructure object was created with
	Config() Config
}

// Provisioner defines a means of creating a cluster from scratch and managing the nodes.
//
// Cluster can be created with a pool of nodes only a subset of which is active at any
// given time. When the cluster is created with capacity bigger than the
// subset of nodes used for installation, then the initially unused nodes can be
// Allocated (and Deallocated) on demand to enable expand/shrink test workflows.
type Provisioner interface {
	// Create provisions a new cluster and returns a reference
	// to the node that can be used to run installation
	//
	// withInstaller specifies if the provisioner should select an installer node.
	// Installer node selection is provisioner-specific
	Create(ctx context.Context, withInstaller bool) (installer Node, err error)
	// Destroy the infrastructures created by Create.
	// After the call to Destroy the provisioner is invalid and no
	// other methods can be used
	Destroy(ctx context.Context) error
	// Connect connects to the node identified with addr and returns
	// a new session object that can be used to execute remote commands
	Connect(addr string) (*ssh.Session, error)
	// Client connects to the node identified with addr and returns
	// a new ssh client that can be used to execute remote commands
	Client(addr string) (*ssh.Client, error)
	// SelectInterface returns the index (in addrs) of network address to use for
	// installation.
	// installerNode should be the result of calling Provisioner.Create
	// addrs is guaranteed to have at least one element
	SelectInterface(installer Node, addrs []string) (int, error)
	// StartInstall initiates installation in the specified session
	StartInstall(session *ssh.Session) error
	// UploadUpdate initiates uploading of new application version
	// in the specified session
	UploadUpdate(session *ssh.Session) error
	// Pool returns a reference to the managing node pool
	NodePool() NodePool
	// InstallerLogPath returns remote path to the installer log file
	InstallerLogPath() string
	// State returns the state of this provisioner
	State() ProvisionerState
}

// NodePool manages node allocation/release for a provisioner
type NodePool interface {
	// Nodes returns all nodes in this pool
	Nodes() []Node
	// AllocatedNodes returns only allocated nodes in this pool
	AllocatedNodes() []Node
	// Node looks up a node given with addr.
	// Returns error if no node matches the specified addr
	Node(addr string) (Node, error)
	// Size returns the number of nodes in this pool
	Size() int
	// SizeAllocated returns the number of allocated nodes in this pool
	SizeAllocated() int
	// Allocate allocates amount new nodes from the pool and returns
	// a slice of allocated nodes
	Allocate(amount int) ([]Node, error)
	// Free releases specified nodes back to the node pool
	Free([]Node) error
}

// Node defines an interface to a remote node
type Node interface {
	fmt.Stringer
	// Addr returns public address of the node
	Addr() string
	// PrivateAddr returns the private address of the node
	PrivateAddr() string
	// Connect connects to this node and returns a new session object
	// that can be used to execute remote commands
	Connect() (*ssh.Session, error)
	// Client connects to this node and returns a new SSH Client object
	// that can be used to execute remote commands
	Client() (*ssh.Client, error)
}

// ExternalStateLoader loads provisioner state from external source
type ExternalStateLoader interface {
	// LoadFromExternalState loads the state from the specified reader r.
	// withInstaller controls whether the installer address information
	// is also retrieved.
	// Returns the installer node if requested.
	LoadFromExternalState(r io.Reader, withInstaller bool) (installer Node, err error)
}

var defaultLogger = log.New()

// Distribute executes the specified command on given nodes
// and waits for execution to complete before returning
func Distribute(command string, nodes ...Node) error {
	log.Infof("running %q on %v", command, nodes)
	errCh := make(chan error, len(nodes))
	wg := sync.WaitGroup{}
	wg.Add(len(nodes))
	for _, node := range nodes {
		go func(node Node) {
			log.Infof("running on %v", node)
			errCh <- Run(node, command, os.Stderr)
			wg.Done()
		}(node)
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
func Run(node Node, command string, w io.Writer) (err error) {
	var session *ssh.Session
	err = wait.Retry(context.TODO(), func() error {
		session, err = node.Connect()
		if err != nil {
			log.Debug(trace.DebugReport(err))
		}
		return trace.Wrap(err)
	})
	if err != nil {
		return trace.Wrap(err)
	}
	return sshutils.RunCommandWithOutput(session, defaultLogger, command, w)
}

// ScpText copies remoteFile from specified node into localFile
func ScpText(node Node, remoteFile string, localFile io.Writer) error {
	return Run(node, fmt.Sprintf("cat %v", remoteFile), localFile)
}
