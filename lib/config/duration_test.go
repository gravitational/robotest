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

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

type TestMessage struct {
	Timeout Timeout `json:"timeout"`
}

func TestJsonDeserialization(t *testing.T) {
	data := []byte(`{"timeout":"30s"}`)
	var msg TestMessage
	err := json.Unmarshal(data, &msg)
	if err != nil {
		t.Error(err)
	}
}

func TestInvalidDeserialization(t *testing.T) {
	data := []byte(`{"timeout":3600}`)
	var msg TestMessage
	err := json.Unmarshal(data, &msg)
	if err == nil {
		t.Errorf("Expected to get an error when deserializing an unqualified integer.")
	}
}

func TestDurationSerialization(t *testing.T) {
	delta, err := time.ParseDuration("24h")
	if err != nil {
		t.Error(err)
	}
	msg := TestMessage{Timeout{delta}}
	data, err := json.Marshal(&msg)
	if err != nil {
		t.Error(err)
	}
	expected := []byte(`{"timeout":"24h0m0s"}`)
	if !bytes.Equal(data, expected) {
		t.Errorf("deserializing 24h, expected %q, got %q", expected, data)
	}

}
