package xlog

import (
	"io/ioutil"

	"github.com/sirupsen/logrus"
)

// ConsoleLogger returns logger which writes everything to file plus console for events above certain level
func ConsoleLogger(consoleLevel logrus.Level) *logrus.Logger {
	log := logrus.New()
	log.Level = logrus.DebugLevel
	log.Out = ioutil.Discard

	consoleLog := logrus.New()
	consoleLog.Level = consoleLevel
	log.Hooks.Add(&consoleHook{consoleLog, consoleLevel})

	return log
}

type consoleHook struct {
	console *logrus.Logger
	level   logrus.Level
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

	switch e.Level {
	case logrus.PanicLevel:
		log.Panic(e.Message)
	case logrus.FatalLevel:
		log.Fatal(e.Message)
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
