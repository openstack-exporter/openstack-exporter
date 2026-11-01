package exporters

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	clientconfig "github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

type timeoutTransport func(*http.Request) (*http.Response, error)

func (f timeoutTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestAuthenticationTimeout(t *testing.T) {
	httpClient := &http.Client{
		Timeout: 25 * time.Millisecond,
		Transport: timeoutTransport(func(r *http.Request) (*http.Response, error) {
			<-r.Context().Done()
			return nil, r.Context().Err()
		}),
	}
	start := time.Now()
	_, err := AuthenticatedClientV2(&clientconfig.ClientOpts{
		EnvPrefix: "TIMEOUT_TEST_",
		AuthType:  clientconfig.AuthV3Password,
		AuthInfo: &clientconfig.AuthInfo{
			AuthURL: "http://identity/v3", Username: "reader", Password: "test", UserDomainName: "Default",
		},
		HTTPClient: httpClient,
	}, nil)
	require.ErrorContains(t, err, "context deadline exceeded")
	require.Less(t, time.Since(start), time.Second)
}

func TestCollectionDeadlineSpansRequests(t *testing.T) {
	requests := 0
	client := &gophercloud.ServiceClient{
		Endpoint: "http://compute/",
		ProviderClient: &gophercloud.ProviderClient{HTTPClient: http.Client{
			Timeout: time.Second,
			Transport: timeoutTransport(func(r *http.Request) (*http.Response, error) {
				requests++
				if requests == 1 {
					return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
				}
				<-r.Context().Done()
				return nil, r.Context().Err()
			}),
		}},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	exporter := &BaseOpenStackExporter{
		Name: "nova", ExporterConfig: ExporterConfig{Prefix: "openstack", ClientV2: client, CollectTimeout: 25 * time.Millisecond}, logger: logger,
	}
	exporter.AddMetric("test", func(ctx context.Context, e *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		for range 2 {
			_, err := e.ClientV2.Get(ctx, e.ClientV2.Endpoint, nil, nil)
			if err != nil {
				return err
			}
			nextDeadline, ok := ctx.Deadline()
			require.True(t, ok)
			require.Equal(t, deadline, nextDeadline)
		}
		return nil
	}, nil, "", nil)
	start := time.Now()
	require.NoError(t, testutil.CollectAndCompare(exporter, strings.NewReader(`
# HELP openstack_nova_up up
# TYPE openstack_nova_up gauge
openstack_nova_up 0
`)))
	require.Equal(t, 2, requests)
	require.Less(t, time.Since(start), time.Second)
}
