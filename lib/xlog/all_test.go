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

package xlog

import (
	"context"
	"testing"

	"github.com/gravitational/trace"

	"github.com/sirupsen/logrus"
)

func TestGcl(t *testing.T) {
	ctx := context.Background()
	client, err := NewGCLClient(ctx, "kubeadm-167321")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	fields := logrus.Fields{"os": "ubuntu", "storage_driver": "devicemapper", "node": "10.0.4.5"}

	log := NewLogger(client, t, fields)

	_, err = NewGCLClient(ctx, "doesnotexist")

	log.WithFields(logrus.Fields{
		"wrapped_no_message":   trace.Wrap(err),
		"wrapped_with_message": trace.Wrap(err, "client error"),
		"multi_wrap":           trace.Wrap(trace.Wrap(err)),
		"errorf":               trace.Errorf("there was an error %v", err),
	}).Error("structured error test")

	short, err := client.Shorten(ctx, `https://console.cloud.google.com/logs/viewer?project=kubeadm-167321&authuser=1&organizationId=419984272859&minLogLevel=0&expandAll=false&resource=global&advancedFilter=resource.type%3D%22global%22%0Alabels.uuid%3D%2231ef6882-0948-4a2c-81df-5bdee81d3c62%22`)
	if err != nil {
		log.WithError(err).Error("Failed to shorten URL")
	} else {
		log.Info(short)
	}
}
