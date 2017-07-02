package xlog

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
)

type TestingHook struct {
	t *testing.T
}

func (hook *TestingHook) Fire(e *logrus.Entry) error {
	hook.t.Log(e.Message, fmt.Sprint(e.Data))
	return nil
}

// Levels returns logging levels supported by logrus
func (hook *TestingHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}

func NewLogger(client *GCLClient, t *testing.T, commonFields logrus.Fields) logrus.FieldLogger {
	log := ConsoleLogger(logrus.InfoLevel)

	if client != nil {
		log.Hooks.Add(client.Hook(t.Name(), commonFields))
	}
	log.Hooks.Add(&TestingHook{t})
	return log
}
