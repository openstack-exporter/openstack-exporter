package clients

import (
	"testing"
)

// RequireLong will ensure long-running tests can run.
func RequireLong(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
}
