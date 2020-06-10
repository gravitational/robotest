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

package framework

import (
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/lib/loc"
	"github.com/gravitational/trace"
)

// TestState represents the state of the test between boostrapping a cluster
// and teardown.
// The state is updated on each in-between test run to sync the provisioner state.
type TestState struct {
	// EntryURL defines the entry point to the application.
	// This can be the address of existing Ops Center or local application endpoint URL
	EntryURL string `json:"ops_url,omitempty"`
	// Application defines the application package to test as retrieved from the wizard
	Application *loc.Locator `json:"application,omitempty"`
	// Login specifies optional login to connect to the EntryURL.
	// Falls back to TestContext.Login if unspecified
	Login *Login `json:"login,omitempty"`
	// ServiceLogin specifies optional service login to connect to the EntryURL.
	ServiceLogin *ServiceLogin `json:"service_login,omitempty"`
	// Bandwagon specifies bandwagon creation details
	Bandwagon *BandwagonConfig `json:"bandwagon,omitempty"`
	// Provisioner defines the provisioner used to create the infrastructure.
	// This can be empty for the automatic provisioner
	Provisioner *Provisioner `json:"provisioner,omitempty"`
	// Onprem defines the provisioner state.
	// The provisioner used is specified by Provisioner.
	// With automatic provisioner, no provisioner state is stored
	ProvisionerState *infra.ProvisionerState `json:"provisioner_state,omitempty"`
	// StateDir specifies the location of temporary state used for a single test run
	// (from bootstrapping to destroy)
	StateDir string `json:"state_dir"`
	// BackupState defines state of backup.
	// Used for backup/restore operations.
	BackupState *BackupState `json:"backup_state,omitempty"`
}

// BackupState defines state of backup.
type BackupState struct {
	// Addr is the address of a node where backup is storing
	Addr string `json:"addr"`
	// Path is an absolute path to the backup file
	Path string `json:"path"`
}

func (r TestState) Validate() error {
	var errors []error
	if r.Provisioner != nil && r.ProvisionerState == nil {
		errors = append(errors, trace.BadParameter("ProvisionerState is required"))
	}
	if r.Provisioner == nil && r.ProvisionerState != nil {
		errors = append(errors, trace.BadParameter("Provisioner is required"))
	}
	if r.StateDir == "" {
		errors = append(errors, trace.BadParameter("StateDir is required"))
	}
	return trace.NewAggregate(errors...)
}
