package gravity

import (
	"context"
	"fmt"
	"net/url"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	"github.com/gravitational/robotest/lib/xlog"

	"github.com/gravitational/trace"

	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

type TestFunc func(c *TestContext, config ProvisionerConfig)

type TestSuite interface {
	// Cancel requests teardown for all subordinate tests
	Cancel(reason string)
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
	// TestStatusPaniced means test function had an unexpected panic
	TestStatusPaniced = "PANICED"
)

// TestStatus represents high level test status on completion
type TestStatus struct {
	UID, SuiteUID string
	Name          string
	Status        string
	LogUrl        string
	Param         interface{}
}

// testRun logically groups multiple test runs for centralized progress and status reporting
type testSuite struct {
	sync.RWMutex

	googleProjectID string
	client          *xlog.GCLClient
	uid             string

	tests     []*TestContext
	scheduled map[string]func(t *testing.T)
	t         *testing.T

	failFast, isFailingFast bool
	ctx                     context.Context
	cancelFn                func()

	logger logrus.FieldLogger
}

// NewRun creates new group run environment
func NewSuite(ctx context.Context, t *testing.T, googleProjectID string, fields logrus.Fields, failFast bool) TestSuite {
	uid := uuid.NewV4().String()
	fields["__suite__"] = uid

	scheduled := map[string]func(t *testing.T){}

	client, err := xlog.NewGCLClient(ctx, googleProjectID)
	logger := xlog.NewLogger(client, t, fields)
	if err != nil {
		logger.WithError(err).Error("cloud logging not available")
	}

	ctx, cancelFn := context.WithCancel(ctx)

	return &testSuite{sync.RWMutex{}, googleProjectID, client, uid,
		[]*TestContext{}, scheduled, t,
		failFast, false, ctx, cancelFn, logger}
}

func (s *testSuite) Logger() logrus.FieldLogger {
	return s.logger
}

// Cancel will request everything to teardown
func (s *testSuite) Cancel(reason string) {
	if s.failingFast() {
		return
	}
	s.Lock()
	s.isFailingFast = true
	s.Unlock()

	s.cancelFn()
	s.Logger().WithField("reason", reason).Warn("test suite canceled")
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
			"resource":  []string{"global"},
			"authuser":  []string{"1"},
			"advancedFilter": []string{
				fmt.Sprintf(`resource.type="global"
labels.__uuid__="%s"
labels.__suite__="%s"
severity>=INFO`, testUID, s.uid)},
		}.Encode()}

	short, err := s.client.Shorten(s.ctx, longUrl.String())
	return short, trace.Wrap(err)
}

func (s *testSuite) wrap(fn TestFunc, cfg ProvisionerConfig, param interface{}) func(t *testing.T) {
	uid := uuid.NewV4().String()

	labels := logrus.Fields{}

	var logLink string
	var err error

	if s.client != nil {
		labels["__uuid__"] = uid
		labels["__suite__"] = s.uid
		labels["__param__"] = param
		logLink, err = s.getLogLink(uid)
		if err != nil {
			s.Logger().WithError(err).Error("Failed to create short log link")
		}
	}

	return func(t *testing.T) {
		ctx, cancelFn := context.WithCancel(s.ctx)
		defer cancelFn()

		cx := &TestContext{
			t:        t,
			name:     cfg.Tag(),
			parent:   ctx,
			timeouts: DefaultTimeouts,
			uid:      uid,
			suite:    s,
			param:    param,
			logLink:  logLink,
			log:      xlog.NewLogger(s.client, t, labels),
		}
		defer func() {
			if r := recover(); r != nil {
				cx.updateStatus(TestStatusPaniced)
				cx.Logger().WithFields(logrus.Fields{
					"stack": debug.Stack(),
					"where": r}).Error("PANIC")
				return
			}
			if s.failingFast() {
				cx.updateStatus(TestStatusCancelled)
				return
			}
			if !t.Failed() {
				cx.updateStatus(TestStatusPassed)
				return
			}
			cx.updateStatus(TestStatusFailed)
			if s.failFast {
				s.Cancel(fmt.Sprintf("test %s failed, FailFast=true, cancelling other", t.Name()))
			}
			return
		}()

		s.Lock()
		s.tests = append(s.tests, cx)
		s.Unlock()

		if logLink != "" {
			cx.log = cx.log.WithField("logs", logLink)
		}

		cx.updateStatus(TestStatusScheduled)
		t.Parallel()
		cx.updateStatus(TestStatusRunning)
		cx.timestamp = time.Now()

		fn(cx, cfg)
	}
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
