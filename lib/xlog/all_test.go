package xlog

import (
	"context"
	"testing"

	"github.com/gravitational/trace"

	"github.com/sirupsen/logrus"
)

type Val struct {
	Val int `json:"val"`
}

func TestGcl(t *testing.T) {
	client, err := NewGCLClient(context.Background(), "kubeadm-167321")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	fields := logrus.Fields{"os": "ubuntu", "storage_driver": "devicemapper", "node": "10.0.4.5"}

	log := NewLogger(client, t, fields)

	log.WithError(trace.Errorf("warning")).Warn("there was an error")
	log.Error("error")
	log.Debug("debug")
	log.Info("info")
	log.WithFields(logrus.Fields{
		"text":   "text val",
		"object": Val{15},
	}).Error("structured error")
}
