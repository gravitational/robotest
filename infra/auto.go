package infra

// autoCluster represents a cluster managed by an active OpsCenter
type autoCluster struct {
	config      Config
	provisioner Provisioner
}

func (r *autoCluster) OpsCenterURL() string { return r.config.OpsCenterURL }
func (r *autoCluster) Config() Config       { return r.config }

func (r *autoCluster) Provisioner() Provisioner {
	return r.provisioner
}

func (r *autoCluster) Close() error {
	return nil
}

func (r *autoCluster) Destroy() error {
	return r.provisioner.Destroy()
}
