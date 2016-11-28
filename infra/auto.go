package infra

// autoCluster represents a cluster managed by an active OpsCenter
// An auto cluster may or may not have a provisioner. When no provisioner
// is specified, the cluster is automatically provisioned
type autoCluster struct {
	config       Config
	provisioner  Provisioner
	opsCenterURL string
}

func (r *autoCluster) OpsCenterURL() string { return r.opsCenterURL }
func (r *autoCluster) Config() Config       { return r.config }

func (r *autoCluster) Provisioner() Provisioner {
	return r.provisioner
}

func (r *autoCluster) Close() error {
	return nil
}

func (r *autoCluster) Destroy() error {
	if r.provisioner != nil {
		return r.provisioner.Destroy()
	}
	return nil
}
