package gravity

import (
	"context"
	"fmt"
	"time"

	"github.com/gravitational/robotest/lib/xlog"
	"github.com/gravitational/trace"

	"cloud.google.com/go/bigquery"
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
	Install, Upgrade, Status, Uninstall, Leave, CollectLogs, WaitForInstaller time.Duration
}

// TestContext aggregates common parameters for better test suite readability
type TestContext struct {
	err       error
	timestamp time.Time
	name      string
	parent    context.Context
	timeouts  OpTimeouts
	log       logrus.FieldLogger
	uid       string
	suite     *testSuite
	param     interface{}
	logLink   string
	status    string
}

// Run allows a running test to spawn a subtest
func (cx *TestContext) Run(fn TestFunc, cfg ProvisionerConfig, param interface{}) {
	t := cx.suite.t
	t.Helper()
	t.Run(cfg.Tag(), cx.suite.wrap(fn, cfg, param))
}

// Context provides a context for a current test run
func (c *TestContext) Context() context.Context {
	return c.parent
}

// Logger returns preconfigured logger for this test
func (c *TestContext) Logger() logrus.FieldLogger {
	return c.log
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

// Checkpoint marks milestone within a test
func (c *TestContext) OK(msg string, err error) {
	now := time.Now()
	elapsed := now.Sub(c.timestamp)
	c.timestamp = now

	fields := logrus.Fields{
		"name":    c.name,
		"elapsed": elapsed.String(),
	}
	if err != nil {
		fields["error"] = err
		c.log.WithFields(fields).Error(msg)
		c.err = trace.Wrap(err)
		panic(msg)
	}
	c.log.WithFields(fields).Info(msg)
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
	case <-c.parent.Done():
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
