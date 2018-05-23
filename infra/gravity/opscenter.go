package gravity

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/gravitational/robotest/lib/defaults"
	"github.com/gravitational/robotest/lib/wait"
	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
)

// getTeleClusterStatus will attempt to get the cluster status from an ops center
// for now, it's just looking for the specific line of output from tele get clusters
// but could parse the entire yaml structure if imported from gravity
func getTeleClusterStatus(ctx context.Context, clusterName string) (string, error) {
	out, err := exec.CommandContext(ctx, "tele", "get", "clusters", clusterName, "--format", "yaml").Output()
	if err != nil {
		if isClusterNotFoundError(out) {
			return "", trace.NotFound("cluster not found")
		}

		logrus.WithError(err).Error("unable to parse tele get clusters: ", string(out))
		return "", trace.WrapWithMessage(err, string(out))
	}

	res, err := parseClusterStatus(out)
	if err != nil {
		logrus.WithError(err).Error("unable to parse tele get clusters: ", string(out))
		return "", trace.WrapWithMessage(err, string(out))
	}
	return res, nil
}

func isClusterNotFoundError(buf []byte) bool {
	if buf == nil {
		return false
	}

	s := string(buf)
	if strings.Contains(s, "cluster") && strings.Contains(s, "not found") {
		return true
	}
	return false
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
func parseClusterStatus(data []byte) (string, error) {
	if len(data) == 0 {
		return "", trace.BadParameter("missing cluster data")
	}

	if isClusterNotFoundError(data) {
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

		ctx, cancel := context.WithTimeout(tc.Context(), defaults.DestroyOpsTimeout)
		defer cancel()

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

		out, err := exec.CommandContext(ctx, "tele", "rm", "cluster", clusterName).CombinedOutput()
		if err != nil {
			return err
		}
		log.Debug("tele rm result: ", string(out))

		retryer := wait.Retryer{
			Delay:    5 * time.Second,
			Attempts: 120,
		}

		err = retryer.Do(ctx, func() (err error) {
			// check provisioning status
			status, err := getTeleClusterStatus(tc.Context(), clusterName)
			if err != nil && trace.IsNotFound(err) {
				// de-provisioning completed
				return nil
			}
			if err != nil {
				return trace.Wrap(err)
			}

			switch status {
			case "uninstalling":
				return trace.Retry(trace.BadParameter("uninstall not complete"), "uninstall not complete")
			default:
				return wait.Abort(trace.BadParameter("unexpected cluster status: %v", status))
			}
		})
		if err != nil {
			return trace.Wrap(err)
		}

		return nil
	}
}
