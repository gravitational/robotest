package infra

type TestDriver interface {
	Install(cluster Infra, installerURL string) error
	Expand(Node) error
	Shrink(Node) error
	// AppUpdate() error
	// Update() error
}
