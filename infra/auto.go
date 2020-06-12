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

package infra

import "context"

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
		return r.provisioner.Destroy(context.TODO())
	}
	return nil
}
