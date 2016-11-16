package infra

import "github.com/gravitational/trace"

func (r *Config) Validate() error {
	var errors []error
	if r.ClusterName == "" {
		errors = append(errors, trace.BadParameter("cluster name is required"))
	}
	if r.OpsCenterURL != "" && len(r.InitialCluster) == 0 {
		errors = append(errors, trace.BadParameter("initial cluster is required for OpsCenterURL"))
	}
	if len(r.InitialCluster) != 0 && r.OpsCenterURL == "" {
		errors = append(errors, trace.BadParameter("OpsCenterURL is required for an existing cluster"))
	}
	return trace.NewAggregate(errors...)

}

type Config struct {
	// ClusterName is the name assigned to the provisioned machines
	ClusterName string `json:"cluster_name" env:"ROBO_CLUSTER_NAME"`
	// OpsCenterURL defines the OpsCenter address.
	// OpsCenter address is optional and is derived automatically
	// when running tests in wizard mode
	OpsCenterURL string `json:"opscenter_url,omitempty"  env:"ROBO_OPS_URL"`
	// InitialCluster sets the node addresses of the existing cluster
	InitialCluster []string `json:"initial_cluster,omitempty" env:"ROBO_INITIAL_CLUSTER"`
	// License defines the license for the installation
	License string `json:"license" env:"ROBO_LICENSE"`
}
