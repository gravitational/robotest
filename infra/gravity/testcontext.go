package gravity

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gravitational/robotest/lib/xlog"

	"cloud.google.com/go/pubsub"
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
	t         *testing.T
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
	cx.t.Run(cfg.Tag(), cx.suite.wrap(fn, cfg, param))
}

// FailNow will interrupt current test
func (cx *TestContext) FailNow() {
	cx.t.FailNow()
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

// OK is equivalent to require.NoError
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
		c.t.FailNow()
	}
	c.log.WithFields(fields).Info(msg)
}

// Require verifies condition is true, fails test otherwise
func (c *TestContext) Require(msg string, condition bool, args ...interface{}) {
	if condition {
		return
	}
	c.log.WithField("args", args).Errorf("failed check: %s", msg)
	c.t.FailNow()
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

type gclMessage struct {
	Status string `json:"status"`
	UUID   string `json:"uuid"`
}

func (c *TestContext) updateStatus(status string) {
	c.status = status

	log := c.Logger().WithFields(logrus.Fields{"param": xlog.ToJSON(c.param), "name": c.name})
	switch c.status {
	case TestStatusScheduled, TestStatusRunning, TestStatusPassed:
		log.Info(c.status)
	default:
		log.Error(c.status)
	}

	client := c.suite.client
	if client == nil {
		return
	}
	data, err := json.Marshal(&gclMessage{status, c.uid})
	if err != nil {
		log.WithError(err).Error("can't json serialize test status")
		return
	}
	res := client.Topic().Publish(client.Context(), &pubsub.Message{Data: data})
	_, err = res.Get(client.Context())
	if err != nil {
		log.WithError(err).Error("failed to report test status due to pubsub error")
	}
}
