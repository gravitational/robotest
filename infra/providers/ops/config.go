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

package ops

// Config specified Ops Center specific parameters
type Config struct {
	// URL to the ops center to use for deployment
	URL string `json:"url" yaml:"url" validate:"required"`
	// OpsKey is the key to connect to the ops center
	OpsKey string `json:"ops_key" yaml:"ops_key" validate:"required"`
	// App is the ops center application to deploy
	App string `json:"app" yaml:"app" validate:"required"`
	// EC2AccessKey http://docs.aws.amazon.com/general/latest/gr/managing-aws-access-keys.html
	EC2AccessKey string `json:"access_key" yaml:"access_key" validate:"required"`
	// EC2SecretKey http://docs.aws.amazon.com/general/latest/gr/managing-aws-access-keys.html
	EC2SecretKey string `json:"secret_key" yaml:"secret_key" validate:"required"`
	// EC2Region specifies the EC2 region to install into
	EC2Region string `json:"region" yaml:"region" validate:"required"`
	// SSHKeyPath specifies the location of the SSH key to use for remote access.
	// Mandatory only with terraform provisioner
	SSHKeyPath string `json:"key_path" yaml:"key_path"`
	// SSHUser defines SSH user used to connect to the provisioned machines
	SSHUser string `json:"ssh_user" yaml:"ssh_user" validate:"required"`
}
