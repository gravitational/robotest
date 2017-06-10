package sshutils

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"strconv"
	"strings"

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
				val, _, err := RunAndParse(ctx, node, "date +%s%3N", nil, parseTime)
				errCh <- err
				valueCh <- val
			}(node)
		}

		values, errors := utils.Collect(ctx, nil, errCh, valueCh)
		if errors != nil {
			return wait.AbortRetry{errors}
		}

		if timeInRange(values) {
			return nil
		}

		return wait.ContinueRetry{fmt.Sprintf("not all system clocks updated with NTP: %v", values)}
	}
}

const (
	maxDelta = 200.0
)

func timeInRange(values []interface{}) bool {
	if len(values) < 2 {
		return true
	}

	d0 := values[0].(float64)
	for _, v := range values[1:] {
		if math.Abs(d0-v.(float64)) > maxDelta {
			return false
		}
	}
	return true
}

func parseTime(r *bufio.Reader) (interface{}, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, trace.Wrap(err)
	}

	ts, err := strconv.ParseFloat(strings.TrimRight(line, "\n"), 64)
	if err != nil {
		return nil, trace.Wrap(err, line)
	}

	io.Copy(ioutil.Discard, r)
	return ts, nil
}
