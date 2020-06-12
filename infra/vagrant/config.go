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

package vagrant

import (
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/trace"
)

// Validate validates the configuration
func (r *Config) Validate() error {
	var errors []error
	if r.ScriptPath == "" {
		errors = append(errors, trace.BadParameter("script path is required"))
	}
	if r.NumNodes <= 0 {
		errors = append(errors, trace.BadParameter("cannot provision %v nodes", r.NumNodes))
	}
	return trace.NewAggregate(errors...)
}

type Config struct {
	infra.Config
	// ScriptPath is the path to the Vagrantfile for provisioning
	ScriptPath string `json:"script_path"`
	// InstallerURL is a path to the installer
	InstallerURL string `json:"installer_url"`
	// NumNodes defines the capacity of the cluster to provision
	NumNodes int `json:"nodes"`
	// DockerDevice block device for docker data - set to /dev/xvdb
	DockerDevice string `json:"docker_device"`
}
