package gravity

import (
	"context"
	"fmt"
	"time"

	"github.com/gravitational/robotest/lib/xlog"

	"cloud.google.com/go/bigquery"
	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
)

const (
	Parallel   = true
	Sequential = false
)

// OpTimeouts defines per-node, per-operation timeouts which would be used to determine
// whether test must be failed
// provisioner has its own timeout / restart logic which is dependant on cloud provider and terraform
type OpTimeouts struct {
	Install          time.Duration
	Upgrade          time.Duration
	Status           time.Duration
	Uninstall        time.Duration
	UninstallApp     time.Duration
	Leave            time.Duration
	CollectLogs      time.Duration
	WaitForInstaller time.Duration
	AutoScaling      time.Duration
}

// TestContext aggregates common parameters for better test suite readability
type TestContext struct {
	err            error
	timestamp      time.Time
	name           string
	ctx            context.Context
	cancel         context.CancelFunc
	timeouts       OpTimeouts
	log            logrus.FieldLogger
	uid            string
	suite          *testSuite
	param          interface{}
	logLink        string
	status         string
	provisionerCfg ProvisionerConfig
	fields         logrus.Fields

	// Context and cancel function for the SSH channel monitor process.
	// Monitor process is usually a long-running process that is active
	// for the lifetime of a single test.
	// Whenever the monitor process is aborted (i.e. interrupted w/o
	// providing exit code), the whole test is aborted and retried
	// (subject to a maximum number of retry attempts)
	monitorCtx    context.Context
	monitorCancel context.CancelFunc
	// preempted indicates that a node part of the cluster that belongs
	// to this context was preempted
	preempted bool
}

// Run allows a running test to spawn a subtest
func (cx *TestContext) Run(fn TestFunc, cfg ProvisionerConfig, param interface{}) {
	t := cx.suite.t
	t.Helper()
	t.Run(cfg.Tag(), cx.suite.wrap(fn, cfg, param))
}

// Context provides a context for a current test run
func (c *TestContext) Context() context.Context {
	return c.ctx
}

// Logger returns preconfigured logger for this test
func (c *TestContext) Logger() logrus.FieldLogger {
	if len(c.fields) == 0 {
		return c.log
	}
	return c.log.WithFields(c.fields)
}

// WithTimeouts returns context
func (c *TestContext) SetTimeouts(tm OpTimeouts) {
	c.timeouts = tm
}

// Failed checks if this test failed
func (c *TestContext) Failed() bool {
	return c.err != nil
}

// Error returns reason this test failed
func (c *TestContext) Error() error {
	return c.err
}

// WithFields assigns additional logging fields to this context
func (c *TestContext) WithFields(fields logrus.Fields) *TestContext {
	c.fields = fields
	return c
}

// OK logs the specified message and error.
// If the error is non-nil, the test is marked failed and aborted
func (c *TestContext) OK(msg string, err error) {
	now := time.Now()
	elapsed := now.Sub(c.timestamp)
	c.timestamp = now
	fields := logrus.Fields{
		"name":    c.name,
		"elapsed": elapsed.String(),
	}

	for name, value := range c.fields {
		fields[name] = value
	}

	if err == nil {
		c.log.WithFields(fields).Info(msg)
		return
	}

	fields["error"] = err
	c.log.WithFields(fields).Error(msg)
	c.err = trace.Wrap(err)
	panic(msg)
}

// Maybe logs the specified message and error if non-nil.
// Does not fail the test
func (c *TestContext) Maybe(msg string, err error) {
	now := time.Now()
	elapsed := now.Sub(c.timestamp)
	c.timestamp = now
	fields := logrus.Fields{
		"name":    c.name,
		"elapsed": elapsed.String(),
	}

	for name, value := range c.fields {
		fields[name] = value
	}

	if err == nil {
		c.log.WithFields(fields).Info(msg)
		return
	}
	fields["error"] = err
	c.log.WithFields(fields).Warn(msg)
}

// FailNow requests this test suite to abort
func (c *TestContext) FailNow() {
	if c.err == nil {
		c.err = fmt.Errorf("request to cancel")
	}
	panic(c.err.Error())
}

// Require verifies condition is true, fails test otherwise
func (c *TestContext) Require(msg string, condition bool, args ...interface{}) {
	if condition {
		return
	}
	c.log.WithField("args", args).Errorf("failed check: %s", msg)
	panic(msg)
}

// Sleep will just sleep with log message
func (c *TestContext) Sleep(msg string, d time.Duration) {
	c.log.Debugf("sleep %v %s...", d, msg)
	select {
	case <-time.After(d):
	case <-c.ctx.Done():
	}
}

func withDuration(d time.Duration, n int) time.Duration {
	return d * time.Duration(n)
}

type progressMessage struct {
	status      string
	suite, uuid string
	name        string
	param       interface{}
}

func (msg progressMessage) Save() (row map[string]bigquery.Value, insertID string, err error) {
	row = make(map[string]bigquery.Value)
	row["ts"] = time.Now()

	// identifiers of specific test and group of tests
	row["uuid"] = msg.uuid
	row["suite"] = msg.suite

	row["name"] = msg.name
	row["status"] = msg.status

	bqParam, ok := msg.param.(bigquery.ValueSaver)
	if !ok {
		return nil, "", trace.BadParameter("param is not bigquery.ValueSaver")
	}
	paramRow, _, err := bqParam.Save()
	if err != nil {
		return nil, "", trace.Wrap(err)
	}
	for k, v := range paramRow {
		row[k] = v
	}
	return row, "", nil
}

func (c *TestContext) updateStatus(status string) {
	c.status = status

	log := c.Logger().WithFields(logrus.Fields{"param": xlog.ToJSON(c.param), "name": c.name})
	switch c.status {
	case TestStatusScheduled, TestStatusRunning:
		log.Info(c.status)
		return
	case TestStatusPassed:
		log.Info(c.status)
	default:
		log.Error(c.status)
	}

	progress := c.suite.progress
	if progress == nil {
		return
	}

	msg := progressMessage{
		status: status,
		uuid:   c.uid,
		suite:  c.suite.uid,
		name:   c.name,
		param:  c.param,
	}
	data, _, err := msg.Save()
	if err != nil {
		log.WithError(err).Error("BQ MSG FAILED")
		return
	} else {
		log.WithField("data", data).Info("BQ SAVE")
	}

	err = progress.Put(c.Context(), msg)
	if err != nil {
		log.WithError(err).Error("BQ status update failed")
	}
}

func (c *TestContext) markPreempted(node Gravity) {
	// Consider the abort to be an indication of node preemption and
	// cancel the test
	c.Logger().Infof("Node %v was stopped/preempted, cancelling test.", node)
	c.preempted = true
	c.cancel()
}
