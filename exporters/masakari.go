package exporters

import (
	"context"
	"log/slog"
	"net/url"
	"strconv"

	gophercloudv2 "github.com/gophercloud/gophercloud/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// masakariPageLimit is the page size requested when listing segments and hosts.
// The Masakari API paginates list responses, so results are fetched page by
// page using the marker of the last-seen item.
const masakariPageLimit = 1000

// MasakariExporter : extends BaseOpenStackExporter
type MasakariExporter struct {
	BaseOpenStackExporter
}

var defaultMasakariMetrics = []Metric{
	{Name: "segment", Labels: []string{"id", "uuid", "name", "description", "recovery_method", "service_type"}, Fn: ListMasakariHosts},
	{Name: "host", Labels: []string{"id", "uuid", "hostname", "failover_segment_id", "failover_segment_name", "type", "control_attributes"}, Fn: nil},
	{Name: "host_on_maintenance", Labels: []string{"uuid", "hostname", "failover_segment_id"}, Fn: nil},
	{Name: "host_reserved", Labels: []string{"uuid", "hostname", "failover_segment_id"}, Fn: nil},
}

// NewMasakariExporter : returns a pointer to MasakariExporter
func NewMasakariExporter(config *ExporterConfig, logger *slog.Logger) (*MasakariExporter, error) {
	exporter := MasakariExporter{
		BaseOpenStackExporter{
			Name:           "masakari",
			ExporterConfig: *config,
			logger:         logger,
		},
	}

	for _, metric := range defaultMasakariMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}

	return &exporter, nil
}

type masakariSegment struct {
	ID             int    `json:"id"`
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	RecoveryMethod string `json:"recovery_method"`
	ServiceType    string `json:"service_type"`
}

type masakariHost struct {
	ID                int    `json:"id"`
	UUID              string `json:"uuid"`
	Hostname          string `json:"name"`
	Type              string `json:"type"`
	ControlAttributes string `json:"control_attributes"`
	Reserved          bool   `json:"reserved"`
	OnMaintenance     bool   `json:"on_maintenance"`
	FailoverSegmentID string `json:"failover_segment_id"`
}

// ListMasakariHosts lists all failover segments and, for each segment, all of
// its hosts. It emits one info metric per segment and per host, plus boolean
// gauges describing the maintenance and reserved state of every host.
func ListMasakariHosts(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	client := exporter.ClientV2

	segments, err := listMasakariSegments(ctx, client)
	if err != nil {
		return err
	}

	for _, segment := range segments {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["segment"].Metric,
			prometheus.GaugeValue, 1.0,
			strconv.Itoa(segment.ID), segment.UUID, segment.Name, segment.Description,
			segment.RecoveryMethod, segment.ServiceType)

		hosts, err := listMasakariHosts(ctx, client, segment.UUID)
		if err != nil {
			return err
		}

		for _, host := range hosts {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["host"].Metric,
				prometheus.GaugeValue, 1.0,
				strconv.Itoa(host.ID), host.UUID, host.Hostname, host.FailoverSegmentID,
				segment.Name, host.Type, host.ControlAttributes)

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["host_on_maintenance"].Metric,
				prometheus.GaugeValue, boolToFloat64(host.OnMaintenance),
				host.UUID, host.Hostname, host.FailoverSegmentID)

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["host_reserved"].Metric,
				prometheus.GaugeValue, boolToFloat64(host.Reserved),
				host.UUID, host.Hostname, host.FailoverSegmentID)
		}
	}

	return nil
}

// masakariListQuery builds the query string used to page through Masakari list
// endpoints. The marker is the UUID of the last item seen on the previous page.
func masakariListQuery(marker string) string {
	v := url.Values{}
	v.Set("limit", strconv.Itoa(masakariPageLimit))
	if marker != "" {
		v.Set("marker", marker)
	}
	return "?" + v.Encode()
}

// listMasakariSegments returns every failover segment, following pagination
// until a short (or empty) page is returned.
func listMasakariSegments(ctx context.Context, client *gophercloudv2.ServiceClient) ([]masakariSegment, error) {
	var all []masakariSegment
	marker := ""

	for {
		var resp struct {
			Segments []masakariSegment `json:"segments"`
		}
		urlStr := client.ServiceURL("segments") + masakariListQuery(marker)
		if _, err := client.Get(ctx, urlStr, &resp, nil); err != nil {
			return nil, err
		}

		all = append(all, resp.Segments...)
		if len(resp.Segments) < masakariPageLimit {
			break
		}
		marker = resp.Segments[len(resp.Segments)-1].UUID
	}

	return all, nil
}

// listMasakariHosts returns every host belonging to the given segment,
// following pagination until a short (or empty) page is returned.
func listMasakariHosts(ctx context.Context, client *gophercloudv2.ServiceClient, segmentUUID string) ([]masakariHost, error) {
	var all []masakariHost
	marker := ""

	for {
		var resp struct {
			Hosts []masakariHost `json:"hosts"`
		}
		urlStr := client.ServiceURL("segments", segmentUUID, "hosts") + masakariListQuery(marker)
		if _, err := client.Get(ctx, urlStr, &resp, nil); err != nil {
			return nil, err
		}

		all = append(all, resp.Hosts...)
		if len(resp.Hosts) < masakariPageLimit {
			break
		}
		marker = resp.Hosts[len(resp.Hosts)-1].UUID
	}

	return all, nil
}

func boolToFloat64(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
