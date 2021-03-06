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

package gce

import "github.com/gravitational/trace"

// Config specifies Google Compute Engine specific parameters
type Config struct {
	// Project name
	// https://cloud.google.com/resource-manager/docs/creating-managing-projects
	Project string `json:"project" yaml:"project" validate:"required"`
	// Credentials names the service account file
	// https://cloud.google.com/compute/docs/api/how-tos/authorization
	Credentials string `json:"credentials" yaml:"credentials" validate:"required"`
	// Region specifies the datacenter region to install into.
	// Can be a comma-separated list of regions.
	// https://cloud.google.com/compute/docs/regions-zones/
	Region string `json:"region,omitempty" yaml:"region"`
	// Zone specifies the datacenter zone to install into.
	// It is the required parameter as it defines the region as well.
	// https://cloud.google.com/compute/docs/regions-zones/
	Zone string `json:"zone,omitempty" yaml:"zone"`
	// VMType specifies the type of machine to provision
	// https://cloud.google.com/compute/docs/machine-types
	VMType string `json:"vm_type" yaml:"vm_type" validate:"required"`
	// SSHUser defines SSH user to connect to the provisioned machines.
	// Required attribute.
	// Will be determined based on selected cloud provder.
	SSHUser string `json:"os_user" yaml:"os_user"`
	// SSHKeyPath specifies the location of the SSH private key for remote access
	SSHKeyPath string `json:"-" yaml:"ssh_key_path" validate:"required"`
	// SSHPublicKeyPath specifies the location of the public SSH key
	SSHPublicKeyPath string `json:"ssh_pub_key_path" yaml:"ssh_pub_key_path" validate:"required"`
	// NodeTag specifies the node tag to use on GCE.
	// Required attribute.
	// Will be computed based on the cluster name during provisioning
	NodeTag string `json:"node_tag" yaml:"node_tag"`
	// VarFilePath is the path to file with custom terraform variables
	VarFilePath string `json:"-" yaml:"var_file_path"`
	// Network specifies the GCP network the nodes will reside upon.
	Network string `json:"network" yaml:"network"`
	// Subnet specifies the GCP subnet the nodes will reside upon.
	Subnet string `json:"subnet" yaml:"subnet"`
}

func (c *Config) CheckAndSetDefaults() error {
	if c == nil {
		return trace.BadParameter("gcp config: configuration is required")
	}
	if c.Project == "" {
		return trace.BadParameter("gcp config: Project is required")
	}
	if c.Credentials == "" {
		return trace.BadParameter("gcp config: Credentials are required")
	}
	if c.SSHUser == "" {
		return trace.BadParameter("gcp config: SSHUser is required")
	}
	if c.SSHKeyPath == "" {
		return trace.BadParameter("gcp config: SSHKeyPath is required")
	}
	if c.SSHPublicKeyPath == "" {
		return trace.BadParameter("gcp config: SSHPublicKeyPath is required")
	}
	if c.VMType == "" {
		return trace.BadParameter("gcp config: VMType is required")
	}
	if c.Network == "" {
		c.Network = "default"
	}
	if c.Subnet == "" {
		c.Subnet = "default"
	}
	return nil
}
