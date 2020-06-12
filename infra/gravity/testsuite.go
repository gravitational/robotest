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

package gravity

import (
	"context"
	"fmt"
	"net/url"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	"github.com/gravitational/robotest/lib/defaults"
	"github.com/gravitational/robotest/lib/wait"
	"github.com/gravitational/robotest/lib/xlog"

	"github.com/cenkalti/backoff"
	"github.com/gravitational/trace"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

type TestFunc func(c *TestContext, config ProvisionerConfig)

type TestSuite interface {
	// Cancel requests teardown for all subordinate tests
	Cancel(reason string, args ...interface{})
	// Schedule adds tests to the plan
	Schedule(fn TestFunc, baseConfig ProvisionerConfig, param interface{})
	// Run executes scheduled (and derived) tests and returns their status
	Run() []TestStatus
	// Logger provides preconfigured logger
	Logger() logrus.FieldLogger
	// Close disposes background resources
	Close()
}

const (
	// TestStatusScheduled means test was scheduled
	TestStatusScheduled = "SCHEDULED"
	// TestStatusRunning means test is running now
	TestStatusRunning = "RUNNING"
	// TestStatusPassed means test successfully passed end to end
	TestStatusPassed = "PASSED"
	// TestStatusFailed means test failed due to test logic not passing
	TestStatusFailed = "FAILED"
	// TestStatusCancelled means test execution was interrupted due to test suite cancellation
	TestStatusCancelled = "CANCELED"
	// TestStatusPanicked means test function had an unexpected panic
	TestStatusPanicked = "PANICKED"
)

// TestStatus represents high level test status on completion
type TestStatus struct {
	UID, SuiteUID string
	Name          string
	Status        string
	LogUrl        string
	Param         interface{}
}

// testSuite logically groups multiple test runs for centralized progress and status reporting
type testSuite struct {
	sync.RWMutex

	googleProjectID string
	client          *xlog.GCLClient
	progress        *xlog.ProgressReporter
	uid             string

	tests     []*TestContext
	scheduled map[string]func(t *testing.T)
	t         *testing.T

	failFast, isFailingFast               bool
	retryAttempts, preemptedRetryAttempts int

	ctx    context.Context
	cancel context.CancelFunc

	logger logrus.FieldLogger
}

// NewRun creates new group run environment
func NewSuite(ctx context.Context, t *testing.T, googleProjectID string, fields logrus.Fields, failFast bool, retryAttempts, preemptedRetryAttempts int) TestSuite {

	uid := uuid.NewV4().String()
	fields["__suite__"] = uid

	scheduled := map[string]func(t *testing.T){}

	client, err := xlog.NewGCLClient(ctx, googleProjectID)
	logger := xlog.NewLogger(client, t, fields)
	if err != nil {
		logger.WithError(err).Error("cloud logging not available")
	}

	progress, err := xlog.NewProgressReporter(ctx, googleProjectID, defaults.BQDataset, defaults.BQTable)
	if err != nil {
		logger.WithError(err).Error("cloud progress reporting not available")
	}

	ctx, cancel := context.WithCancel(ctx)

	return &testSuite{
		RWMutex:                sync.RWMutex{},
		googleProjectID:        googleProjectID,
		client:                 client,
		progress:               progress,
		uid:                    uid,
		scheduled:              scheduled,
		t:                      t,
		failFast:               failFast,
		retryAttempts:          retryAttempts,
		preemptedRetryAttempts: preemptedRetryAttempts,
		ctx:                    ctx,
		cancel:                 cancel,
		logger:                 logger,
	}
}

func (s *testSuite) Logger() logrus.FieldLogger {
	return s.logger
}

// Cancel will request everything to teardown
func (s *testSuite) Cancel(reason string, args ...interface{}) {
	if s.failingFast() {
		return
	}
	s.Lock()
	s.isFailingFast = true
	s.Unlock()

	s.cancel()
	s.Logger().WithField("reason", fmt.Sprintf(reason, args...)).Warn("Test suite canceled.")
}

func (s *testSuite) failingFast() bool {
	s.RLock()
	defer s.RUnlock()

	return s.isFailingFast
}

func (s *testSuite) Close() {
	if s.client != nil {
		s.client.Close()
		s.client = nil
	}
}

func (s *testSuite) Schedule(fn TestFunc, cfg ProvisionerConfig, param interface{}) {
	s.scheduled[cfg.Tag()] = s.wrap(fn, cfg, param)
}

func (s *testSuite) getLogLink(testUID string) (string, error) {
	longUrl := url.URL{
		Scheme: "https",
		Host:   "console.cloud.google.com",
		Path:   "/logs/viewer",
		RawQuery: url.Values{
			"project":   []string{s.googleProjectID},
			"expandAll": []string{"false"},
			"authuser":  []string{"1"},
			"advancedFilter": []string{
				fmt.Sprintf(`resource.type="project"
labels.__uuid__="%s"
labels.__suite__="%s"
severity>=INFO`, testUID, s.uid)},
		}.Encode(),
	}

	// Google URL shortener has been discontinued.
	// See https://developers.googleblog.com/2018/03/transitioning-google-url-shortener.html for details.
	// TODO(dmitri): decide whether it would make sense to migrate to Firebase Dynamic Links as a replacement.
	// For now, return full URLs
	return longUrl.String(), nil
}

func (s *testSuite) wrap(fn TestFunc, baseConfig ProvisionerConfig, param interface{}) func(t *testing.T) {
	return func(t *testing.T) {
		t.Helper()
		t.Parallel()

		b := newPreemptiveBackoff(s.retryAttempts, s.preemptedRetryAttempts)
		try := 0
		err := wait.RetryWithInterval(s.ctx, b, func() error {
			t.Helper()

			try++
			cfg := baseConfig
			if try > 1 {
				cfg = baseConfig.WithTag(fmt.Sprintf("T%d", try))
				s.Logger().Warnf("Retrying %q (%d/%d).",
					cfg.Tag(), b.numTries, b.maxTries)
			}

			testCtx, err := s.runTestFunc(t, fn, cfg, param)
			if err == nil {
				return nil
			}

			if testCtx.preempted {
				b.nextPreempted()
			} else {
				b.next()
			}

			s.Logger().WithError(err).Warnf("Test %q completed with error.", cfg.Tag())

			if s.failingFast() {
				t.Skip("context cancelled")
				return nil
			}

			if trace.IsBadParameter(err) {
				// this usually means either a panic inside test,
				// or bad configuration parameters passed to it
				// there's no reason to retry it
				return &backoff.PermanentError{Err: trace.Wrap(err)}
			}

			// an error will be retried
			return trace.Wrap(err)
		}, s.Logger())

		if err == nil {
			return
		}

		if s.failFast {
			s.Cancel("Test %s failed, FailFast=true, cancelling other.", t.Name())
		}

		t.Error(trace.Wrap(err))
	}
}

func (s *testSuite) runTestFunc(t *testing.T, testFunc TestFunc, cfg ProvisionerConfig, param interface{}) (testCtx *TestContext, err error) {
	uid := uuid.NewV4().String()
	labels := logrus.Fields{}
	var logLink string

	if s.client != nil {
		labels["__uuid__"] = uid
		labels["__suite__"] = s.uid
		labels["__param__"] = param
		labels["__name__"] = cfg.Tag()
		logLink, err = s.getLogLink(uid)
		if err != nil {
			s.Logger().WithError(err).Error("Failed to create short log link.")
		}
	}

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	defer monitorCancel()

	testCtx = &TestContext{
		name:     cfg.Tag(),
		ctx:      ctx,
		cancel:   cancel,
		timeouts: DefaultTimeouts,
		uid:      uid,
		suite:    s,
		param:    param,
		logLink:  logLink,
		log: xlog.NewLogger(s.client, t, labels).WithFields(logrus.Fields{
			"name": cfg.Tag(),
		}),
		monitorCtx:    monitorCtx,
		monitorCancel: monitorCancel,
	}

	defer func() {
		r := recover()
		if r == nil {
			testCtx.updateStatus(TestStatusPassed)
			return
		}

		if s.failingFast() {
			testCtx.updateStatus(TestStatusCancelled)
			return
		}

		if testCtx.Failed() {
			testCtx.updateStatus(TestStatusFailed)
			err = testCtx.Error()
			return
		}

		// genuine panic by test itself, not after cx.OK()
		// usually that is a logical error in a test itself
		// there is no reason to retry it
		testCtx.updateStatus(TestStatusPanicked)
		testCtx.Logger().WithFields(
			logrus.Fields{
				"stack": string(debug.Stack()),
				"where": r,
			},
		).Error("Panic in test.")
		err = trace.BadParameter("panic inside test - aborted")
	}()

	if logLink != "" {
		testCtx.log = testCtx.log.WithField("logs", logLink)
	}

	s.Lock()
	s.tests = append(s.tests, testCtx)
	s.Unlock()
	testCtx.updateStatus(TestStatusRunning)

	testCtx.timestamp = time.Now()
	testFunc(testCtx, cfg)

	return testCtx, nil
}

// Run executes all tests in this suite and returns test results
func (s *testSuite) Run() []TestStatus {
	s.t.Run("run", func(t *testing.T) {
		for tag, fn := range s.scheduled {
			t.Run(tag, fn)
		}
	})

	status := []TestStatus{}
	for _, test := range s.tests {
		status = append(status, TestStatus{
			Name:     test.name,
			Status:   test.status,
			Param:    test.param,
			UID:      test.uid,
			SuiteUID: test.suite.uid,
			LogUrl:   test.logLink,
		})
	}
	return status
}

func newPreemptiveBackoff(maxTries, maxPreempted int) *preemptiveBackoff {
	b := wait.NewUnlimitedExponentialBackoff()
	return &preemptiveBackoff{
		delegate:     b,
		maxTries:     maxTries,
		maxPreempted: maxPreempted,
	}
}

func (r *preemptiveBackoff) NextBackOff() time.Duration {
	if r.numTries >= r.maxTries {
		return backoff.Stop
	}
	return r.delegate.NextBackOff()
}

func (r *preemptiveBackoff) Reset() {
	r.numTries = 0
	r.numPreempted = 0
	r.delegate.Reset()
}

func (r *preemptiveBackoff) nextPreempted() {
	r.numPreempted += 1
	if r.numPreempted >= r.maxPreempted {
		r.numTries = r.maxTries
	}
}

func (r *preemptiveBackoff) next() {
	r.numTries += 1
	r.numPreempted = 0
}

// preemptiveBackoff implements a backoff.BackOff that takes
// node preemptions into account.
// The interval has a retry counter which is bumped by 1
// each time next() is invoked until it exceeds the specified
// maximum at which point the interval is stopped.
// If a node is preempted, preemption counter is bumped by 1
// each time nextPreempted() is invoked until it exceeds the
// specified maximum at which point the interval is stopped.
//
// Increasing the preemption counter does not increase the regular
// counter. If the regular counter is increased, the preemption
// counter is reset.
type preemptiveBackoff struct {
	delegate                   backoff.BackOff
	numPreempted, maxPreempted int
	numTries, maxTries         int
}
