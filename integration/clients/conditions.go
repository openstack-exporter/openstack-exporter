package clients

import (
	acceptance "github.com/gophercloud/gophercloud/v2/internal/acceptance/clients"
	"testing"
)

// RequireLong skips cloud integration tests in short mode.
func RequireLong(t *testing.T) {
	t.Helper()
	acceptance.RequireLong(t)
}
