package framework

import "time"

// Extensions groups configuration options for individual test steps.
// TODO: we need to come up with a way to support configuration for arbitraty test steps.
// This is just to jump-start the solution to the most obvious pain points
type Extensions struct {
	// InstallTimeout specifies the total time to wait for install operation to complete.
	// Defaults to ui/defaults.InstallTimeout if unspecified
	InstallTimeout duration `json:"install_timeout" yaml:"install_timeout" env:"ROBO_INSTALL_TIMEOUT"`
}

// duration aliases time.Duration to support JSON/Env serialisation
type duration time.Duration

// Duration returns this duration as time.Duration
func (r duration) Duration() time.Duration {
	return time.Duration(r)
}

// SetEnv interprets data as time.Duration.
// SetEnv implements configure.EnvSetter
func (r *duration) SetEnv(data string) error {
	d, err := time.ParseDuration(data)
	if err != nil {
		return err
	}
	*r = duration(d)
	return nil
}

// UnmarshalText interprets data as time.Duration.
// UnmarshalText implements encoding.TextUnmarshaler
func (r *duration) UnmarshalText(data []byte) error {
	return r.SetEnv(string(data))
}
