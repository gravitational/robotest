package gravity

import (
	"context"
	"testing"
	"time"

	"github.com/gravitational/robotest/lib/xlog"

	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

type TestFunc func(c TestContext, config ProvisionerConfig)

const (
	Parallel   = true
	Sequential = false
)

func Wrap(fn TestFunc, ctx context.Context, cfg ProvisionerConfig, client *xlog.GCLClient, fields logrus.Fields) func(*testing.T) {
	return func(t *testing.T) {
		uid := uuid.NewV4().String()

		labels := logrus.Fields{}
		for k, v := range fields {
			labels[k] = v
		}
		labels["tag"] = cfg.Tag()
		labels["os"] = cfg.os
		labels["storage"] = cfg.storageDriver
		labels["uuid"] = uid

		logger := xlog.NewLogger(client, t, labels)
		cx := TestContext{t, ctx, DefaultTimeouts, logger, client}
		cx.reportScheduled()
		t.Parallel()
		cx.reportStarted()
		fn(cx, cfg)
		cx.reportCompleted(!t.Failed())
	}
}

func (cx TestContext) Run(fn TestFunc, cfg ProvisionerConfig, fields logrus.Fields) {
	cx.t.Run(cfg.Tag(), Wrap(fn, cx.Context(), cfg, cx.client, fields))
}

func (cx TestContext) FailNow() {
	cx.t.FailNow()
}

// OpTimeouts define per-node, per-operation timeouts which would be used to determine
// whether test must be failed
// provisioner has its own timeout / restart logic which is dependant on cloud provider and terraform
type OpTimeouts struct {
	Install, Status, Uninstall, Leave, CollectLogs time.Duration
}

var DefaultTimeouts = OpTimeouts{
	Install:     time.Minute * 15,
	Uninstall:   time.Minute * 5,
	Status:      time.Minute * 30, // sufficient for failover procedures
	Leave:       time.Minute * 15,
	CollectLogs: time.Minute * 7,
}

// TestContext aggregates common parameters for better test suite readability
type TestContext struct {
	t        *testing.T
	parent   context.Context
	timeouts OpTimeouts
	log      logrus.FieldLogger
	client   *xlog.GCLClient
}

func (c TestContext) Context() context.Context {
	return c.parent
}

func (c TestContext) Logger() logrus.FieldLogger {
	return c.log
}

func (c TestContext) WithTimeouts(tm OpTimeouts) TestContext {
	c.timeouts = tm
	return c
}

// OK is equivalent to require.NoError
func (c TestContext) OK(msg string, err error) {
	if err != nil {
		c.log.WithFields(logrus.Fields{"type": "assert", "error": err}).Fatal(msg)
		c.t.FailNow()
	}
	c.log.WithField("type", "assert").Info(msg)
}

// Require verifies condition is true, fails test otherwise
func (c TestContext) Require(msg string, condition bool, args ...interface{}) {
	if condition {
		return
	}
	c.log.WithField("args", args).Fatal(msg)
	c.t.FailNow()
}

// Sleep will just sleep with log message
func (c TestContext) Sleep(msg string, d time.Duration) {
	c.log.Debugf("Sleep %v %s...", d, msg)
	select {
	case <-time.After(d):
	case <-c.parent.Done():
	}
}

func withDuration(d time.Duration, n int) time.Duration {
	return d * time.Duration(n)
}

func (c TestContext) reportScheduled()             {}
func (c TestContext) reportStarted()               {}
func (c TestContext) reportCompleted(success bool) {}
