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
