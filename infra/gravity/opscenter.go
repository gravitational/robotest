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

package gravity

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// getTeleClusterStatus will attempt to get the cluster status from an ops center
// for now, it's just looking for the specific line of output from tele get clusters
// but could parse the entire yaml structure if imported from gravity
func getTeleClusterStatus(clusterName string) (string, error) {
	out, err := exec.Command("tele", "get", "clusters", clusterName, "--format", "yaml").Output()
	if err != nil {
		logrus.WithError(err).Error("unable to parse tele get clusters: ", string(out))
		return "", trace.WrapWithMessage(err, string(out))
	}

	res, err := parseClusterStatus(clusterName, out)
	if err != nil {
		logrus.WithError(err).Error("unable to parse tele get clusters: ", string(out))
		return "", trace.WrapWithMessage(err, string(out))
	}
	return res, nil
}

// ClusterV2 spec is a minimal copy of the Cluster definition from the gravitational/gravity project
// Unneeded fields have been removed, and only required fields are included in the local copy
// https://github.com/gravitational/gravity/blob/20cfcef8d50ab403f0a9452376ccd52e145ae90c/lib/storage/cluster.go#L65
type ClusterV2 struct {
	// Spec contains cluster specification
	Spec ClusterSpecV2 `json:"spec" yaml:"spec"`
}

// ClusterSpecV2 is cluster V2 specification from gravitational/gravity project
// it is a minimal copy of only needed fields
type ClusterSpecV2 struct {
	// Status is a cluster status, initialized for existing clusters only
	Status string `json:"status,omitempty" yaml:"status,omitempty"`
}

// parseClusterStatus will attempt to unmarshal the cluster status from tele get clusters output
func parseClusterStatus(clusterName string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", trace.BadParameter("missing cluster data")
	}

	if strings.Contains(string(data), fmt.Sprintf("cluster %v not found", clusterName)) {
		return "", trace.NotFound("cluster not found")
	}

	cluster := ClusterV2{}
	err := yaml.Unmarshal(data, &cluster)
	if err != nil {
		return "", trace.Wrap(err)
	}

	return cluster.Spec.Status, nil
}

// generateClusterConfig will generate a cluster configuration for the ops center based
// on the built in template
func generateClusterConfig(cfg ProvisionerConfig, clusterName string) (string, error) {
	template, err := template.New("cluster").Parse(`
kind: cluster
version: v2
metadata:
  labels:
    Name: {{ .ClusterName }}
  name: {{ .ClusterName }}
spec:
  app: {{ .Cfg.Ops.App }}
  aws:
    keyName: ops
    region: {{ .Cfg.Ops.EC2Region }}
  nodes:
  - profile: node
    count: {{ .Cfg.NodeCount }}
    instanceType: c4.2xlarge
  provider: aws`)

	if err != nil {
		return "", trace.Wrap(err)
	}
	buf := &bytes.Buffer{}
	err = template.Execute(buf, struct {
		Cfg         ProvisionerConfig
		ClusterName string
	}{
		cfg,
		clusterName,
	})
	if err != nil {
		return "", trace.Wrap(err)
	}
	return buf.String(), nil
}

// DestroyOpsFn will destroy the cluster by making a request to the ops center to de-provision the cluster
func (c ProvisionerConfig) DestroyOpsFn(tc *TestContext, clusterName string) func() error {
	return func() error {
		log := tc.Logger().WithFields(logrus.Fields{
			"cluster": clusterName,
		})

		// if the TestContext has an error, it means this robotest run failed
		if tc.Context().Err() != nil {
			log.WithError(tc.Context().Err()).Info("error caught in testing")

			if !policy.DestroyOnFailure {
				log.Info("skipped destroy due to error")
				return trace.Wrap(tc.Context().Err())
			}
		} else {
			if !policy.DestroyOnSuccess {
				log.Info("skipped destroy on success due to policy")
				return nil
			}
		}

		log.Info("destroying cluster")

		out, err := exec.Command("tele", "rm", "cluster", clusterName).CombinedOutput()
		if err != nil {
			return err
		}
		log.Debug("tele rm result: ", string(out))

		// monitor the cluster until it's gone
		timeout := time.After(DefaultTimeouts.Uninstall)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				return trace.LimitExceeded("clusterDestroy timeout exceeded")
			case <-ticker.C:
				// check provisioning status
				status, err := getTeleClusterStatus(clusterName)
				if err != nil && trace.IsNotFound(err) {
					// de-provisioning completed
					return nil
				}
				if err != nil {
					return trace.Wrap(err)
				}

				switch status {
				case "uninstalling":
					// we're still uninstalling, just continue the loop
				default:
					return trace.BadParameter("unexpected cluster status: %v", status)
				}
			}
		}
	}
}
