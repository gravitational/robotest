package infra

import "github.com/gravitational/trace"

func (r *Config) Validate() error {
	if r.ClusterName == "" {
		return trace.BadParameter("cluster name is required")
	}
	return nil
}

type Config struct {
	// ClusterName is the name assigned to the provisioned machines
	ClusterName string `json:"cluster_name" `
}

// ProvisionerState defines the state configuration for a cluster
// provisioned with a specific provisioner
type ProvisionerState struct {
	// Dir defines the location where provisioner stores state
	Dir string `json:"state_dir"`
	// InstallerAddr is the address of the installer node
	InstallerAddr string `json:"installer_addr,omitempty"`
	// Nodes is a list of all nodes in the cluster
	Nodes []StateNode `json:"nodes"`
	// Allocated defines the allocated subset
	Allocated []string `json:"allocated_nodes"`
	// LoadBalancerAddr defines the DNS name of the load balancer
	LoadBalancerAddr string `json:"loadbalancer"`
}

// StateNode describes a single cluster node
type StateNode struct {
	// Addr is the address of this node
	Addr string `json:"addr"`
	// KeyPath defines the location of the SSH key
	KeyPath string `json:"key_path"`
}
