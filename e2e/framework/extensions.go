/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import "time"

// Extensions groups configuration options for individual test steps.
// TODO: we need to come up with a way to support configuration for arbitraty test steps.
// This is just to jump-start the solution to the most obvious pain points
type Extensions struct {
	// InstallTimeout specifies the total time to wait for install operation to complete.
	// Defaults to ui/defaults.InstallTimeout if unspecified
	InstallTimeout duration `json:"install_timeout" yaml:"install_timeout" `
	// BackupConfig defines configuration for Backup/Restore operations
	BackupConfig *BackupConfig `json:"backup_config" yaml:"backup_config"`
}

// BackupConfig defines configuration for Backup/Restore operations
type BackupConfig struct {
	Addr string `json:"addr" yaml:"addr" `
	// BackupPath defines path where Backup will be stored. Path should be absolute.
	// Also this path used for restore operation on node.
	Path string `json:"path" yaml:"path" `
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
