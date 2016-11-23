package framework

import (
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/trace"
)

type TestState struct {
	// OpsCenterURL defines the Ops Center address this infrastructure is
	// communicating with.
	// In case of the wizard mode, this is the actual Ops Center created by the wizard
	// which was not available upfront
	OpsCenterURL string `json:"ops_url"`
	// Provisioner defines the provisioner used to create the infrastructure.
	// This can be empty for the automatic provisioner
	Provisioner provisionerType `json:"provisioner"`
	// Onprem defines the provisioner state.
	// The provisioner used is specified by Provisioner.
	// Only a single state is active at any time. With automatic provisioner,
	// no provisioner state is stored
	ProvisionerState infra.ProvisionerState `json:"provisioner_state"`
}

func (r TestState) Validate() error {
	var errors []error
	if r.OpsCenterURL == "" {
		errors = append(errors, trace.BadParameter("Ops Center URL is required"))
	}
	return trace.NewAggregate(errors...)
}