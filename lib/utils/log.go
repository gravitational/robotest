package utils

import (
	"fmt"
	"log"
	"testing"
)

func init() {
	log.SetFlags(0)
}

type LogFnType func(format string, args ...interface{})

func Logf(t *testing.T, prefix string) LogFnType {
	return func(format string, args ...interface{}) {
		t.Logf(format, args...)
		log.Printf(fmt.Sprintf("[%s] %s", prefix, format), args...)
	}
}
