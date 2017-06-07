package sshutils

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strconv"
	"time"

	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"
)

// WaitTimeSync will ensure time is synchronized between the nodes provided
// otherwise an installation might fail
// we do not try to cross check times between nodes, but rather check their system clock drift
// from NTP source
func WaitTimeSync(ctx context.Context, nodes []SshNode) error {
	if len(nodes) < 2 {
		return nil
	}

	return wait.Retry(ctx, checkTimeInSync(ctx, nodes))
}

func checkTimeInSync(ctx context.Context, nodes []SshNode) func() error {
	return func() error {
		errCh := make(chan error, len(nodes))
		valueCh := make(chan interface{}, len(nodes))

		for _, node := range nodes {
			go func(node SshNode) {
				val, _, err := RunAndParse(ctx, node, "chronyc tracking", nil, parseChronyc)
				errCh <- err
				valueCh <- val
			}(node)
		}

		values, errors := utils.Collect(ctx, nil, errCh, valueCh)
		if errors != nil {
			return wait.AbortRetry{errors}
		}

		if inSyncWithNTP(values) {
			return nil
		}

		return wait.ContinueRetry{fmt.Sprintf("not all system clocks updated with NTP: %v", values)}
	}
}

const (
	maxDelta = time.Millisecond * 300
)

func inSyncWithNTP(values []interface{}) bool {
	for _, v := range values {
		d := v.(time.Duration)
		if d >= maxDelta {
			return false
		}
	}
	return true
}

var reSystemTime = regexp.MustCompile(`(System time)\s+\: ([\d\.]+) seconds .*`)

func parseChronyc(r *bufio.Reader) (interface{}, error) {
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			return nil, trace.Errorf("reached end of output, no `System time` report found")
		}
		if err != nil {
			return nil, trace.Wrap(err)
		}

		vars := reSystemTime.FindStringSubmatch(line)
		if len(vars) == 3 {
			io.Copy(ioutil.Discard, r)
			d, err := strconv.ParseFloat(vars[2], 64)
			if err != nil {
				return nil, trace.Wrap(err, line)
			}
			return time.Second * time.Duration(d), nil
		}
	}

	return nil, trace.Errorf("failed to parse `chronyc tracking` output")
}
