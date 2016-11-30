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
	// OpsCenterURL defines the Ops Center address this infrastructure is
	// communicating with.
	// In case of the wizard mode, this is the actual Ops Center created by the wizard
	// which was not available upfront.
	// This can be different from the Ops Center configured in TestContext which always
	// refers to the originating Ops Center
	OpsCenterURL string `json:"ops_url"`
	// Application defines the application package to test as retrieved from the wizard
	Application *loc.Locator `json:"application,omitempty"`
	// Provisioner defines the provisioner used to create the infrastructure.
	// This can be empty for the automatic provisioner
	Provisioner provisionerType `json:"provisioner,omitempty"`
	// Onprem defines the provisioner state.
	// The provisioner used is specified by Provisioner.
	// With automatic provisioner, no provisioner state is stored
	ProvisionerState *infra.ProvisionerState `json:"provisioner_state,omitempty"`
	// StateDir specifies the location of temporary state used for a single test run
	// (from bootstrapping to destroy)
	StateDir string `json:"state_dir"`
}

func (r TestState) Validate() error {
	var errors []error
	if r.OpsCenterURL == "" {
		errors = append(errors, trace.BadParameter("Ops Center URL is required"))
	}
	if r.Provisioner != "" && r.ProvisionerState == nil {
		errors = append(errors, trace.BadParameter("ProvisionerState is required"))
	}
	if r.Provisioner == "" && r.ProvisionerState != nil {
		errors = append(errors, trace.BadParameter("Provisioner is required"))
	}
	if r.StateDir == "" {
		errors = append(errors, trace.BadParameter("StateDir is required"))
	}
	return trace.NewAggregate(errors...)
}
