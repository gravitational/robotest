package e2e

import (
	"testing"

	"github.com/gravitational/robotest/e2e/framework"
)

func init() {
	framework.ConfigureFlags()
}

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}
