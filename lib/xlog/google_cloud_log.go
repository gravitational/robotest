package xlog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"runtime"

	"github.com/gravitational/trace"

	cl "cloud.google.com/go/logging"
	"cloud.google.com/go/pubsub"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
)

const (
	statusTopic  = "robotest"
	logViewerUrl = "https://console.cloud.google.com/logs/viewer"
	maxStack     = 10
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
	shortenerClient *http.Client
	gclClient       *cl.Client
	pubsubClient    *pubsub.Client
	topic           *pubsub.Topic
	ctx             context.Context
}

func (client *GCLClient) Close() {
	client.gclClient.Close()
	client.topic.Stop()
	client.pubsubClient.Close()
}

type GCLHook struct {
	log          *cl.Logger
	commonFields logrus.Fields
}

// NewGCLClient tries to establish connection to google cloud logger using default authentication method and project ID
func NewGCLClient(ctx context.Context, projectID string) (client *GCLClient, err error) {
	if projectID == "" {
		return nil, trace.Errorf("no cloud logging project ID provided")
	}

	client = &GCLClient{ctx: ctx}

	// URL shortener API
	client.shortenerClient, err = google.DefaultClient(ctx, urlShortenerScope)
	if err != nil {
		return nil, trace.Wrap(err, "Google OAuth failed")
	}

	// Google Cloud Logger API
	client.gclClient, err = cl.NewClient(ctx, projectID)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = client.gclClient.Ping(ctx)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	// Pub/Sub API
	client.pubsubClient, err = pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	client.topic = client.pubsubClient.Topic(statusTopic)
	ok, err := client.topic.Exists(ctx)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	if ok {
		return client, nil
	}

	client.topic, err = client.pubsubClient.CreateTopic(ctx, statusTopic)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return client, nil
}

// Context returns context instance this client was initialized with
// as it may survive local function context which is i.e. cancelled or timed out
func (c *GCLClient) Context() context.Context {
	return c.ctx
}

// Hook returns logrus log hook
func (c *GCLClient) Hook(name string, fields logrus.Fields) *GCLHook {
	labels := map[string]string{}
	for k, v := range fields {
		switch v.(type) {
		case string:
			labels[k] = v.(string)
		default:
			labels[k] = ToJSON(v)
		}
	}

	return &GCLHook{
		c.gclClient.Logger(name, cl.CommonLabels(labels)),
		fields}
}

func ToJSON(obj interface{}) string {
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("%v", obj)
	}
	return string(data)
}

// Topic returns google pub/sub topic for test result status reporting
func (c GCLClient) Topic() *pubsub.Topic {
	return c.topic
}

// Fire fires the event to the GCL
func (hook *GCLHook) Fire(e *logrus.Entry) error {
	severity, ok := levelMap[e.Level]
	if !ok {
		severity = cl.Default
	}

	p := e.WithFields(logrus.Fields{"stack": where(maxStack), "message": e.Message}).Data
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

var exclude = regexp.MustCompile(`github\.com/sirupsen/logrus|/usr/local/go/src|robotest/infra/gravity/testcontext\.go|robotest/infra/gravity/testsuite\.go`)

func where(max int) (stack []string) {
	for i := 3; i <= 10 && len(stack) < max; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		if !exclude.MatchString(file) {
			stack = append(stack, fmt.Sprintf("%s:%d", shortPath(file), line))
		}
	}
	return stack
}

var shortPackage = regexp.MustCompile(`(\/[a-zA_Z\_]+){1,3}\.go$`)

func shortPath(p string) string {
	if s := shortPackage.FindString(p); s != "" {
		return s
	}
	return p
}
