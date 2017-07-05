package xlog

import (
	"io/ioutil"

	"github.com/sirupsen/logrus"
)

// ConsoleLogger returns logger which writes everything to file plus console for events above certain level
func ConsoleLogger(consoleLevel logrus.Level, stackDepth int) *logrus.Logger {
	log := logrus.New()
	log.Level = logrus.DebugLevel
	log.Out = ioutil.Discard

	consoleLog := logrus.New()
	consoleLog.Level = consoleLevel
	log.Hooks.Add(&consoleHook{consoleLog, consoleLevel, stackDepth})

	return log
}

type consoleHook struct {
	console    *logrus.Logger
	level      logrus.Level
	stackDepth int
}

func (hook *consoleHook) Fire(e *logrus.Entry) error {
	if e.Level > hook.level {
		return nil
	}

	var log logrus.FieldLogger
	if e.Data != nil {
		log = hook.console.WithFields(e.Data)
	} else {
		log = hook.console
	}

	if hook.stackDepth > 0 {
		log = log.WithField("where", where(hook.stackDepth))
	}

	switch e.Level {
	case logrus.PanicLevel:
		log.WithField("message", e.Message).Error("**** log.Panic() SHOULD NOT BE INVOKED  ****")
	case logrus.FatalLevel:
		log.WithField("message", e.Message).Error("**** log.Fatal() SHOULD NOT BE INVOKED  ****")
	case logrus.ErrorLevel:
		log.Error(e.Message)
	case logrus.WarnLevel:
		log.Warn(e.Message)
	case logrus.InfoLevel:
		log.Info(e.Message)
	case logrus.DebugLevel:
		log.Debug(e.Message)
	}

	return nil
}

// Levels returns logging levels supported by logrus
func (hook *consoleHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}
