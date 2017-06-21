package gravity

import (
	"context"
	"regexp"
	"strings"

	"github.com/gravitational/trace"
)

var reIpAddr = regexp.MustCompile(`(([0-9]{1,3})\.([0-9]{1,3})\.([0-9]{1,3})\.([0-9]{1,3}))`)

// ResolveInPlanet will launch DNS resolution inside Planet container
func ResolveInPlanet(ctx context.Context, g Gravity, name string) (string, error) {
	if name == "" {
		return "", trace.BadParameter("should provide name to resolve")
	}

	addr, err := g.RunInPlanet(ctx, "/usr/bin/dig", []string{"+short", name})
	addr = reIpAddr.FindString(addr)
	if addr == "" {
		return "", trace.NotFound("no records for %s", name)
	}

	return addr, trace.Wrap(err)
}

var reSpaces = regexp.MustCompile(`\s+`)

// GetGravitySiteNodes will parse `kubectl get pods` output to figure out which nodes run gravity site master
func GetGravitySiteNodes(ctx context.Context, g Gravity) (master string, other []string, err error) {
	out, err := g.RunInPlanet(ctx, "/usr/bin/kubectl", []string{"get", "pods", "-o=wide", "--namespace=kube-system"})
	if err != nil {
		return "", nil, trace.Wrap(err)
	}

	other = []string{}
	for i, line := range strings.Split(out, "\n") {
		if i == 0 || line == "" {
			continue
		}
		// NAME                             READY     STATUS     RESTARTS   AGE       IP            NODE
		// bandwagon-66870618-g9b55         1/1       Running    0          11h       10.244.25.3   10.40.2.5
		vals := reSpaces.Split(line, -1)
		if len(vals) != 7 {
			return "", nil, trace.Errorf("unexpected string %q", line)
		}

		if !strings.HasPrefix(vals[0], "gravity-site") {
			continue
		}

		if vals[1] == "1/1" {
			master = vals[6]
		} else {
			other = append(other, vals[6])
		}
	}

	return master, other, nil
}
