package gravity

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"testing"

	"github.com/gravitational/robotest/lib/xlog"

	"github.com/gravitational/trace"

	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

type TestFunc func(c TestContext, config ProvisionerConfig)

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

// TestStatus represents high level test status on completion
type TestStatus struct {
	UID, SuiteUID string
	Name          string
	Failed        bool
	LogUrl        string
	Param         interface{}
}

// testRun logically groups multiple test runs for centralized progress and status reporting
type testSuite struct {
	sync.Mutex

	googleProjectID string
	client          *xlog.GCLClient
	uid             string

	tests     []TestContext
	scheduled map[string]func(t *testing.T)
	t, runner *testing.T

	ctx      context.Context
	cancelFn func()

	logger logrus.FieldLogger
}

// NewRun creates new group run environment
func NewSuite(ctx context.Context, t *testing.T, googleProjectID string, fields logrus.Fields) TestSuite {
	uid := uuid.NewV4().String()
	fields["__uuid__"] = uid

	scheduled := map[string]func(t *testing.T){}

	ctx, cancelFn := context.WithCancel(ctx)

	client, err := xlog.NewGCLClient(ctx, googleProjectID)
	logger := xlog.NewLogger(client, t, fields)
	if err != nil {
		logger.WithError(err).Error("cloud logging not available")
	}

	return &testSuite{sync.Mutex{}, googleProjectID, client, uid,
		[]TestContext{}, scheduled, t, nil,
		ctx, cancelFn, logger}
}

func (s *testSuite) Logger() logrus.FieldLogger {
	return s.logger
}

// Cancel will request everything to teardown
func (s *testSuite) Cancel(reason string) {
	s.cancelFn()
	s.Logger().WithField("reason", reason).Debug("test suite canceled")
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
labels.__suite__="%s"`, testUID, s.uid)},
		}.Encode()}

	short, err := s.client.Shorten(s.ctx, longUrl.String())
	return short, trace.Wrap(err)
}

func (s *testSuite) wrap(fn TestFunc, cfg ProvisionerConfig, param interface{}) func(t *testing.T) {
	uid := uuid.NewV4().String()

	labels := logrus.Fields{}
	labels["__tag__"] = cfg.Tag()
	labels["__os__"] = cfg.os
	labels["__storage__"] = cfg.storageDriver

	var logLink string
	var err error

	if s.client != nil {
		labels["__uuid__"] = uid
		labels["__suite__"] = s.uid
		logLink, err = s.getLogLink(uid)
		if err != nil {
			s.Logger().WithError(err).Error("Failed to create short log link")
		}
	}

	return func(t *testing.T) {
		cx := TestContext{
			t:        t,
			name:     cfg.Tag(),
			parent:   s.ctx,
			timeouts: DefaultTimeouts,
			uid:      uid,
			suite:    s,
			param:    param,
			logLink:  logLink,
			log:      xlog.NewLogger(s.client, t, labels).WithField("name", cfg.Tag()),
		}
		defer func() {
			if t.Failed() {
				cx.log.Error("failed")
			} else {
				cx.log.Info("passed")
			}
		}()

		s.Lock()
		s.tests = append(s.tests, cx)
		s.Unlock()

		if logLink != "" {
			cx.log = cx.log.WithField("logs", logLink)
		}

		cx.log.Infof("scheduled")
		t.Parallel()
		cx.log.Infof("started")

		fn(cx, cfg)
	}
}

// Run executes all tests in this suite and returns test results
func (s *testSuite) Run() []TestStatus {
	s.t.Run("run", func(t *testing.T) {
		s.runner = t
		for tag, fn := range s.scheduled {
			t.Run(tag, fn)
		}
	})

	status := []TestStatus{}
	for _, test := range s.tests {
		status = append(status, TestStatus{
			Name:     test.name,
			Failed:   test.t.Failed(),
			Param:    test.param,
			UID:      test.uid,
			SuiteUID: test.suite.uid,
			LogUrl:   test.logLink,
		})
	}
	return status
}
