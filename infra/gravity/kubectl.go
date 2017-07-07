package gravity

import (
	"context"
	"strings"

	"github.com/gravitational/robotest/lib/wait"

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
	args := []string{
		"get", "pods", "-n", namespace,
		`-ojsonpath='{range .items[*]}{.metadata.name},{.status.conditions[?(@.type=="Ready")].status},{.status.hostIP}{"\n"}{end}'`,
	}
	if label != "" {
		args = append(args, "-l", label)
	}
	out, err := g.RunInPlanet(ctx, "/usr/bin/kubectl", args...)

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

	// FIXME: https://github.com/gravitational/gravity/issues/2408
	if strings.HasPrefix(out, "Error") {
		return trace.Errorf(out)
	}

	// wait for the pod to disappear
	err = wait.Retry(ctx, func() error {
		pods, err := KubectlGetPods(ctx, g, namespace, "")
		if err != nil {
			return wait.Abort(err)
		}

		for _, p := range pods {
			if p.Name == pod {
				return wait.Continue("pod is still present")
			}
		}

		return nil
	})

	return trace.Wrap(err)
}
