package clients

import (
	"net/http"

	"github.com/gophercloud/utils/v2/client"
)

// newLoggingRoundTripper returns an http.RoundTripper that logs all HTTP
// requests and responses using the gophercloud utils logger. Sensitive
// headers and credential fields are automatically redacted.
func newLoggingRoundTripper() http.RoundTripper {
	return &client.RoundTripper{
		Rt: &http.Transport{},
	}
}
