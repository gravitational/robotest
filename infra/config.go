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
	"encoding/json"

	"github.com/gravitational/trace"
)

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
	Dir string `json:"state_dir,omitempty"`
	// InstallerAddr is the address of the installer node
	InstallerAddr string `json:"installer_addr,omitempty"`
	// Nodes is a list of all nodes in the cluster
	Nodes []StateNode `json:"nodes"`
	// Allocated defines the allocated subset
	Allocated []string `json:"allocated_nodes"`
	// Specific defines provisioner-specific state
	Specific json.Marshaler
}

// StateNode describes a single cluster node
type StateNode struct {
	// Addr is the address of this node
	Addr string `json:"addr"`
	// KeyPath defines the location of the SSH key
	KeyPath string `json:"key_path,omitempty"`
}
