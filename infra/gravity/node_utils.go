package gravity

import (
	"context"
	"regexp"

	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"
)

var reIpAddr = regexp.MustCompile(`(([0-9]{1,3})\.([0-9]{1,3})\.([0-9]{1,3})\.([0-9]{1,3}))`)

// ResolveInPlanet will launch DNS resolution inside Planet container
func ResolveInPlanet(ctx context.Context, g Gravity, name string) (string, error) {
	if name == "" {
		return "", trace.BadParameter("should provide name to resolve")
	}

	addr, err := g.RunInPlanet(ctx, "/usr/bin/dig", "+short", name)
	addr = reIpAddr.FindString(addr)
	if addr == "" {
		return "", trace.NotFound("no records for %s", name)
	}

	return addr, trace.Wrap(err)
}

// RelocateClusterMaster will check which node currently runs gravity-site master
// and will try to evict it from that node so that it'll get picked up by some other
func RelocateClusterMaster(ctx context.Context, g Gravity) error {
	return wait.Retry(ctx, func() error { return doRelocate(ctx, g) })
}

func doRelocate(ctx context.Context, g Gravity) error {
	pods, err := KubectlGetPods(ctx, g, kubeSystemNS, appGravityLabel)
	if err != nil {
		return wait.Abort(trace.Wrap(err))
	}

	var master *Pod
	for _, pod := range pods {
		if pod.Ready {
			master = &pod
			break
		}
	}
	if master == nil {
		return wait.Abort(trace.NotFound("no current cluster master: %+v", pods))
	}

	if err = KubectlDeletePod(ctx, g, kubeSystemNS, master.Name); err != nil {
		return wait.Abort(trace.Wrap(err, "removing pod %s", master.Name))
	}

	var newMaster *Pod
	// wait for relocation to complete
	err = wait.Retry(ctx, func() error {
		pods, err := KubectlGetPods(ctx, g, kubeSystemNS, appGravityLabel)
		if err != nil {
			return wait.Abort(err)
		}

		for _, pod := range pods {
			if pod.Ready {
				newMaster = &pod
				return nil
			}
		}

		return wait.Continue("waiting for gravity-site master to be ready")
	})

	if err != nil {
		return wait.Abort(err)
	}

	if newMaster == nil {
		return wait.Abort(trace.Errorf("new master was not located"))
	}

	if newMaster.NodeIP == master.NodeIP {
		return wait.Continue(
			"new master %+v was elected on same node as old %+v", newMaster, master)
	}

	return nil
}
