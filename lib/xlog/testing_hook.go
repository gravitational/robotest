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
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
)

type TestingHook struct {
	t *testing.T
}

func (hook *TestingHook) Fire(e *logrus.Entry) error {
	hook.t.Helper()
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

// NewLogger returns logger which also prints everything to console
func NewLogger(client *GCLClient, t *testing.T, commonFields logrus.Fields) logrus.FieldLogger {
	consoleLevel := logrus.InfoLevel
	consoleStack := 1
	if client == nil {
		consoleLevel = logrus.DebugLevel
		consoleStack = 3
	}

	log := ConsoleLogger(consoleLevel, consoleStack)

	if client != nil {
		log.Hooks.Add(client.Hook(t.Name(), commonFields))
	} else {
		log.Hooks.Add(&TestingHook{t})
	}
	return log
}
