package vagrant

import (
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/trace"
)

// Validate validates the configuration
func (r *Config) Validate() error {
	var errors []error
	if r.InstallerURL == "" {
		errors = append(errors, trace.BadParameter("installer URL is required"))
	}
	if r.ScriptPath == "" {
		errors = append(errors, trace.BadParameter("script path is required"))
	}
	return trace.NewAggregate(errors...)
}

type Config struct {
	infra.Config
	// ScriptPath is the path to the Vagrantfile for provisioning
	ScriptPath string `json:"script_path" env:"ROBO_SCRIPT_PATH"`
	// InstallerURL is a path to the installer
	InstallerURL string `json:"installer_url" env:"ROBO_INSTALLER_URL"`
	// Nodes is the number of initially active nodes.
	// This can be less than the number of provisioned nodes (e.g. from Vagrantfile).
	// The unallocated nodes can then be used for expand/shrink tests.
	Nodes int `json:"nodes" env:"ROBO_NODES"`
}
