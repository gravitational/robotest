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

package config

/*
 The tests within this file provide minimal testing for Robotest's test configuration DSL,
 and also serve as examples of how custom test parameters can be declared, validated, and
 and have defaults provided.
*/

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/gravitational/robotest/infra/gravity"
	"github.com/gravitational/trace"
)

// defaultTimeout is meaningless test data, it could be any duration
var defaultTimeout = 5 * time.Second

// testParam is an artificial struct similar to suite/sanity/install.go:installParam.
type testParam struct {
	UID       uint             `json:"uid"`
	Username  string           `json:"user"`
	Operation *nestedTestParam `json:"operation"`
}

// CheckAndSetDefaults provides some basic default logic validation for testParam.
//
// This is provided as an example of what sort of custom validation can be done
// by fulfilling the defaulter interface.
func (r *testParam) CheckAndSetDefaults() error {
	var expected string
	switch r.UID {
	case 0:
		expected = "root"
	case 1:
		expected = "daemon"
	default:
		return trace.BadParameter("unknown UID %v", r.UID)
	}
	if r.Username == "" {
		r.Username = expected
	} else {
		if r.Username != expected {
			return trace.BadParameter("username %q does not match UID %v", r.Username, r.UID)
		}
	}
	if r.Operation != nil {
		if err := r.Operation.CheckAndSetDefaults(); err != nil {
			return trace.Wrap(err)
		}
	}
	return nil
}

// nestedTestParam is intended to be a struct within a struct.

// This demonstrates validation and defaults for potentially nil sub parameters.
type nestedTestParam struct {
	Name    string   `json:"name" validate:"required"`
	Timeout *Timeout `json:"timeout"`
}

func (r *nestedTestParam) CheckAndSetDefaults() error {
	if r.Timeout == nil {
		r.Timeout = &Timeout{defaultTimeout}
	} else {
		if r.Timeout.Duration < 0 {
			return trace.BadParameter("timeout must be >= 0")
		}
	}
	if r.Name == "" {
		return trace.BadParameter("operation name must be specified")
	}
	return nil
}

// exampleTest is an example of a test generator that robotest could run, utilizing
// testParam and nestedTestParam
func exampleTest(p interface{}) (gravity.TestFunc, error) {
	param := p.(testParam)
	return func(g *gravity.TestContext, cfg gravity.ProvisionerConfig) {
		out, _ := json.Marshal(param)
		fmt.Println(string(out))
	}, nil
}

func TestCompleteParamValidation(t *testing.T) {
	data := []byte(`{"uid":1,"user":"daemon","operation":{"name":"start","timeout":"30s"}}`)
	var p testParam
	if err := json.Unmarshal(data, &p); err != nil {
		t.Error(err)
	}
	if err := checkAndSetDefaults(&p); err != nil {
		t.Error(err)
	}
	expected := testParam{
		UID:       1,
		Username:  "daemon",
		Operation: &nestedTestParam{Name: "start", Timeout: &Timeout{30 * time.Second}},
	}
	if diff := cmp.Diff(expected, p); diff != "" {
		t.Errorf("param mismatch (-want +got):\n%s", diff)
	}
}

func TestPartialParamValidation(t *testing.T) {
	data := []byte(`{"uid":1}`)
	var p testParam
	if err := json.Unmarshal(data, &p); err != nil {
		t.Error(err)
	}
	if err := checkAndSetDefaults(&p); err != nil {
		t.Error(err)
	}
	expected := testParam{UID: 1, Username: "daemon"}
	if diff := cmp.Diff(expected, p); diff != "" {
		t.Errorf("param mismatch (-want +got):\n%s", diff)
	}
}

func TestInvalidUIDParamValidation(t *testing.T) {
	data := []byte(`{"uid":1000,"user":"centos"}`)
	var p testParam
	if err := json.Unmarshal(data, &p); err != nil {
		t.Error(err)
	}
	err := checkAndSetDefaults(&p)
	if err == nil {
		t.Error("expected an error")
	}
	if !trace.IsBadParameter(err) {
		t.Errorf("expected a bad parameter error, got %v", err)
	}
}

func TestNilableNestedParamDefault(t *testing.T) {
	data := []byte(`{"uid":1,"operation":{"name":"stop"}}`)
	var p testParam
	if err := json.Unmarshal(data, &p); err != nil {
		t.Error(err)
	}
	if err := checkAndSetDefaults(&p); err != nil {
		t.Error(err)
	}
	expected := testParam{
		UID:       1,
		Username:  "daemon",
		Operation: &nestedTestParam{Name: "stop", Timeout: &Timeout{defaultTimeout}},
	}
	if diff := cmp.Diff(expected, p); diff != "" {
		t.Errorf("param mismatch (-want +got):\n%s", diff)
	}
}

func TestMakeFunction(t *testing.T) {
	var defaults testParam
	data := `{"uid":0,"operation":{"name":"stop"}}`
	entry, err := makeFunction(exampleTest, data, defaults)
	if err != nil {
		t.Error(err)
	}
	expected := testParam{
		UID:       0,
		Username:  "root",
		Operation: &nestedTestParam{Name: "stop", Timeout: &Timeout{defaultTimeout}},
	}
	if diff := cmp.Diff(expected, entry.Param); diff != "" {
		t.Errorf("param mismatch (-want +got):\n%s", diff)
	}
	// run the following to make sure it doesn't panic
	testFunc := entry.TestFunc
	testFunc(&gravity.TestContext{}, gravity.ProvisionerConfig{})

}

func TestMakeFunctionNoData(t *testing.T) {
	var defaults testParam
	var data string
	_, err := makeFunction(exampleTest, data, defaults)
	if err == nil {
		t.Error("expected an error")
	}
	if !trace.IsBadParameter(err) {
		t.Errorf("expected a bad parameter error, got %v", err)
	}

}
