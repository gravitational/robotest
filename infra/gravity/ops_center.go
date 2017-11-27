package gravity

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
)

var (
	errClusterNotFound = errors.New("cluster not found")
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

	res, err := parseClusterStatus(clusterName, bytes.NewReader(out))
	if err != nil {
		logrus.WithError(err).Error("unable to parse tele get clusters: ", string(out))
		return "", trace.WrapWithMessage(err, string(out))
	}
	return res, nil
}

// parseClusterStatus will look at the command output, and search for the string "status: abc" and return
// the "abc" portion.
func parseClusterStatus(clusterName string, rd io.Reader) (string, error) {
	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		trimmed := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(trimmed, "status") {
			// we found the status line
			split := strings.Split(trimmed, ": ")
			if len(split) < 2 {
				return "", trace.BadParameter("invalid status line: %v", trimmed)
			}
			return split[1], nil
		}

		if strings.Contains(trimmed, fmt.Sprintf("cluster %v not found", clusterName)) {
			return "", trace.Wrap(errClusterNotFound)
		}
	}

	return "", trace.BadParameter("invalid input")
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

			if policy.DestroyOnFailure == false {
				log.Info("skipped destroy due to error")
				return trace.Wrap(tc.Context().Err())
			}
		} else {
			if policy.DestroyOnSuccess == false {
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
		tick := time.Tick(5 * time.Second)

		for {
			select {
			case <-timeout:
				return trace.LimitExceeded("clusterDestroy timeout exceeded")
			case <-tick:
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
