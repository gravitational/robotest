package gravity

import (
	"context"
	"strings"

	"github.com/gravitational/trace"
)

type Pod struct {
	Name   string
	Ready  bool
	NodeIP string
}

const (
	kubeSystemNS    = "kube-system"
	appGravityLabel = "app=gravity-site"
)

func KubectlGetPods(ctx context.Context, g Gravity, namespace, label string) ([]Pod, error) {
	out, err := g.RunInPlanet(ctx, "/usr/bin/kubectl", "get", "pods",
		"-n", namespace, "-l", label,
		`-ojsonpath='{range .items[*]}{.metadata.name},{.status.conditions[?(@.type=="Ready")].status},{.status.hostIP}{"\n"}{end}'`)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	pods := []Pod{}

	for _, line := range strings.Split(out, "\n") {
		v := strings.Split(line, ",")
		if line == "" {
			continue
		}
		if len(v) != 3 {
			return nil, trace.Errorf("unexpected string %q", line)
		}

		pods = append(pods, Pod{Name: v[0], Ready: v[1] == "True", NodeIP: v[2]})
	}

	return pods, nil
}

func KubectlDeletePod(ctx context.Context, g Gravity, namespace, pod string) error {
	out, err := g.RunInPlanet(ctx, "/usr/bin/kubectl", "delete", "po", "-n", namespace, pod)
	if err != nil {
		return trace.Wrap(err)
	}

	// FIXME: currently we don't get back exit codes from planet enter thus relying on adhoc mechanism
	if strings.HasPrefix(out, "Error") {
		return trace.Errorf(out)
	}

	return nil
}
