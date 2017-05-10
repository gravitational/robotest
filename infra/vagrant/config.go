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
}
