package infra

import "github.com/gravitational/trace"

func (r *Config) Validate() error {
	if r.ClusterName == "" {
		return trace.BadParameter("cluster name is required")
	}
	return nil

}

type Config struct {
	// ClusterName is the name assigned to the provisioned machines
	ClusterName string `json:"cluster_name" env:"ROBO_CLUSTER_NAME"`
	// License defines the license for the installation
	License string `json:"license" env:"ROBO_LICENSE"`
}
