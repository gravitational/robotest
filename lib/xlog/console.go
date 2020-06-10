/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	consoleLog.Formatter = &logrus.TextFormatter{DisableTimestamp: true}
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
