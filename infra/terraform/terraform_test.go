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

package terraform

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gravitational/robotest/infra/providers/gce"
)

func TestConvertConfigToTerraformVars(t *testing.T) {
	gceConfig := gce.Config{
		Credentials:      "/robotest/gce-creds.json",
		VMType:           "excellent",
		SSHUser:          "ubuntu",
		SSHPublicKeyPath: "/robotest/.ssh/robo.pub",
		SSHKeyPath:       "/robotest/.ssh/robo",
		NodeTag:          "unittest",
	}
	cfg := Config{
		CloudProvider: "gce",
		GCE:           &gceConfig,
		OS:            "ubuntu",
		ScriptPath:    "/robotest/assets/terraform/gce",
		NumNodes:      3,
		InstallerURL:  "s3://hub.gravitational.io/gravity/oss/app/telekube/7.0.0/linux/x86_64/telekube-7.0.0-linux-x86_64.tar",
		DockerDevice:  "/dev/xvdb",
		VarFilePath:   "/robotest/custom-vars.json",
	}
	err := cfg.Validate()
	if err != nil {
		t.Error(err)
	}

	configMap, err := configToTerraformVars(cfg)
	if err != nil {
		t.Error(err)
	}
	expected := make(map[string]interface{})
	expected["credentials"] = "/robotest/gce-creds.json"
	expected["nodes"] = 3
	expected["vm_type"] = "excellent"
	expected["os"] = "ubuntu"
	expected["os_user"] = "ubuntu"
	expected["ssh_pub_key_path"] = "/robotest/.ssh/robo.pub"
	expected["node_tag"] = "unittest"

	b, err := json.Marshal(configMap)
	if err != nil {
		t.Error(err)
	}
	e, err := json.Marshal(expected)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(b, e) {
		t.Errorf("\ngot:\t\t%q\nexpected:\t%q", b, e)
	}
}

func TestConvertConfigToTerraformVarsOptionalValues(t *testing.T) {
	gceConfig := gce.Config{
		Region:           "us-west1",
		Zone:             "us-west-1-b",
		Project:          "unittesting",
		Credentials:      "/robotest/gce-creds.json",
		VMType:           "excellent",
		SSHUser:          "ubuntu",
		SSHPublicKeyPath: "/robotest/.ssh/robo.pub",
		SSHKeyPath:       "/robotest/.ssh/robo",
		NodeTag:          "unittest",
	}
	cfg := Config{
		CloudProvider: "gce",
		GCE:           &gceConfig,
		OS:            "ubuntu",
		ScriptPath:    "/robotest/assets/terraform/gce",
		NumNodes:      3,
		InstallerURL:  "s3://hub.gravitational.io/gravity/oss/app/telekube/7.0.0/linux/x86_64/telekube-7.0.0-linux-x86_64.tar",
		DockerDevice:  "/dev/xvdb",
		VarFilePath:   "/robotest/custom-vars.json",
	}
	err := cfg.Validate()
	if err != nil {
		t.Error(err)
	}

	configMap, err := configToTerraformVars(cfg)
	if err != nil {
		t.Error(err)
	}

	expected := make(map[string]interface{})
	expected["region"] = "us-west1"
	expected["zone"] = "us-west-1-b"
	expected["project"] = "unittesting"
	expected["credentials"] = "/robotest/gce-creds.json"
	expected["nodes"] = 3
	expected["vm_type"] = "excellent"
	expected["os"] = "ubuntu"
	expected["os_user"] = "ubuntu"
	expected["ssh_pub_key_path"] = "/robotest/.ssh/robo.pub"
	expected["node_tag"] = "unittest"

	b, err := json.Marshal(configMap)
	if err != nil {
		t.Error(err)
	}
	e, err := json.Marshal(expected)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(b, e) {
		t.Errorf("\ngot:\t\t%q\nexpected:\t%q", b, e)
	}
}
