package functional

import (
	"fmt"
	"log"
	"testing"

	sshutils "github.com/gravitational/robotest/lib/ssh"
)

func init() {
	log.SetFlags(0)
}

func Logf(t *testing.T, prefix string) sshutils.LogFnType {
	return func(format string, args ...interface{}) {
		t.Logf(format, args...)
		log.Printf(fmt.Sprintf("[%s %s] %s", t.Name(), prefix, format), args...)
	}
}
