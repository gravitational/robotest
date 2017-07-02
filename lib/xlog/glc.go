package xlog

import (
	"context"
	"fmt"
	"regexp"
	"runtime"

	"github.com/gravitational/trace"

	cl "cloud.google.com/go/logging"
	"github.com/sirupsen/logrus"
)

var levelMap map[logrus.Level]cl.Severity = map[logrus.Level]cl.Severity{
	logrus.PanicLevel: cl.Emergency,
	logrus.FatalLevel: cl.Critical,
	logrus.ErrorLevel: cl.Error,
	logrus.WarnLevel:  cl.Warning,
	logrus.InfoLevel:  cl.Info,
	logrus.DebugLevel: cl.Debug,
}

type GCLClient struct {
	client *cl.Client
}

func (client *GCLClient) Close() {
	client.client.Close()
}

type GCLHook struct {
	log          *cl.Logger
	commonFields logrus.Fields
}

func NewGCLClient(ctx context.Context, projectID string) (client *GCLClient, err error) {
	if projectID == "" {
		return nil, trace.Errorf("no cloud logging project ID provided")
	}

	client = &GCLClient{}

	client.client, err = cl.NewClient(ctx, projectID)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = client.client.Ping(ctx)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return client, nil
}

func (c *GCLClient) Hook(name string, fields logrus.Fields) *GCLHook {
	labels := map[string]string{}
	for k, v := range fields {
		labels[k] = fmt.Sprintf("%v", v)
	}

	return &GCLHook{
		c.client.Logger(name, cl.CommonLabels(labels)),
		fields}
}

// Fire fires the event to the GCL
func (hook *GCLHook) Fire(e *logrus.Entry) error {
	severity, ok := levelMap[e.Level]
	if !ok {
		severity = cl.Default
	}

	p := e.WithFields(logrus.Fields{"stack": where(), "message": e.Message}).Data
	for key, _ := range hook.commonFields {
		delete(p, key)
	}

	hook.log.Log(cl.Entry{
		Payload:  p,
		Severity: severity})

	return nil
}

// Levels returns logging levels supported by logrus
func (hook *GCLHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}

var exclude = regexp.MustCompile(`github\.com\/sirupsen\/logrus|\/usr\/local\/go\/src`)

func where() (stack []string) {
	for i := 3; i <= 10; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		if !exclude.MatchString(file) {
			stack = append(stack, fmt.Sprintf("%s:%d", file, line))
		}
	}
	return stack
}
