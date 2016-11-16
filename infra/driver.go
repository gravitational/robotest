package infra

type TestDriver interface {
	Install(cluster Infra, installerURL string) error
	// Expand(Infra) error
	// Shrink(Infra) error
	// AppUpdate() error
	// Update() error
}
