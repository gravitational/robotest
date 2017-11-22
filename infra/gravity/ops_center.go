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
	out, err := exec.Command("tele", "get", "clusters", clusterName).Output()
	if err != nil {
		return "", err
	}

	return parseClusterStatus(clusterName, bytes.NewReader(out))
}

func parseClusterStatus(clusterName string, rd io.Reader) (string, error) {
	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		trimmed := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(trimmed, "status") {
			// we found the status line
			split := strings.Split(trimmed, ": ")
			if len(split) < 2 {
				return "", trace.Wrap(fmt.Errorf("failed to parse status line: %v", trimmed))
			}
			return split[1], nil
		}

		if strings.Contains(trimmed, fmt.Sprintf("cluster %v not found", clusterName)) {
			return "", trace.Wrap(errClusterNotFound)
		}
	}

	return "", trace.Wrap(errors.New("unable to parse tele output"))
}

// generateClusterConfig will generate the specified cluster configuration based
// on the built in template, and the provided provisioner configuration
func generateClusterDefn(cfg ProvisionerConfig) (string, error) {
	template, err := template.New("cluster").Parse(`
kind: cluster
version: v2
metadata:
  labels:
    Name: {{ .Ops.ClusterName }}
  name: {{ .Ops.ClusterName }}
  spec:
    app: {{ .Ops.App }}
    aws:
      keyName: ops
      region: {{ .Ops.Region }}
    nodes:
    - profile: node
      count: {{ .NodeCount }}
      instanceType: c4.large
    provider: aws`)

	if err != nil {
		return "", trace.Wrap(err)
	}
	buf := &bytes.Buffer{}
	err = template.Execute(buf, cfg)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// DestroyOpsFn will delete the provisioned cluster
func (c ProvisionerConfig) DestroyOpsFn(tc *TestContext) func() error {
	return func() error {
    log := tc.Logger().WithFields(logrus.Fields{
			"cluster": c.Ops.ClusterName,
		})

    // if the TestContext has an error, it means this robotest run failed
    if tc.Context().Err() != nil {
			log.WithError(tc.Context().Err()).Info("error caught in testing")

			if policy.DestroyOnFailure == false {
				log.Info("skipped destroy due to error")
				return trace.Wrap(tc.Context().Err())
			}
		} else {
			if !policy.DestroyOnSuccess {
				log.Info("skipped destroy on success due to policy")
			}
		}

    // Destroy the cluster
		log.Info("destroying cluster")

    out, err := exec.Command("tele", "rm", "cluster", c.Ops.ClusterName).Output()
		if err != nil {
			return err
		}
    log.Info("tele rm result: ", string(out))

		// monitor the cluster until it's gone
		timeout := time.After(cloudInitTimeout)
		tick := time.Tick(5 * time.Second)

		for {
			select {
			case <-timeout:
				return errors.New("clusterDestroy timeout exceeded")
			case <-tick:
				// check provisioning status
				status, err := getTeleClusterStatus(c.Ops.ClusterName)
				if err != nil && err.Error() == errClusterNotFound.Error() {
					// de-provisioning completed
          return nil
				}
        if err != nil {
          return trace.Wrap(err)
        }

				switch status {
				case "uninstalling":
					// we're still installing, just continue the loop
				default:
					return trace.Wrap(fmt.Errorf("unexpected cluster status: %v", status))
				}
			}
		}
	}
}
