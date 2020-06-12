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

package sshutils

import (
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"math"
	"strconv"

	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/robotest/lib/wait"

	"github.com/gravitational/trace"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// SshNode is ssh node client and logger
type SshNode struct {
	Client *ssh.Client
	Log    logrus.FieldLogger
}

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

// CheckTimeSync verifies current time is in sync across nodes
func CheckTimeSync(ctx context.Context, nodes []SshNode) error {
	if len(nodes) < 2 {
		return nil
	}

	err := checkTimeInSync(ctx, nodes)()
	return trace.Wrap(err)
}

func checkTimeInSync(ctx context.Context, nodes []SshNode) func() error {
	return func() error {
		errCh := make(chan error, len(nodes))
		valueCh := make(chan interface{}, len(nodes))

		for _, node := range nodes {
			go func(node SshNode) {
				var ts float64
				err := RunAndParse(ctx, node.Client, node.Log, "date +%s%3N", nil, parseTime(&ts))
				errCh <- err
				valueCh <- ts
			}(node)
		}

		values, errors := utils.Collect(ctx, nil, errCh, valueCh)
		if errors != nil {
			return wait.AbortRetry{Err: errors}
		}

		if timeInRange(values) {
			return nil
		}

		return wait.Continue("not all system clocks updated with NTP: %v", values)
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

func parseTime(ts *float64) OutputParseFn {
	return func(r *bufio.Reader) error {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			var err error
			*ts, err = strconv.ParseFloat(scanner.Text(), 64)
			_, _ = io.Copy(ioutil.Discard, r)
			return trace.ConvertSystemError(err)
		}
		if err := scanner.Err(); err != nil {
			return trace.ConvertSystemError(err)
		}
		return trace.BadParameter("did not read any lines")
	}
}
