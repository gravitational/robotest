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
