package exporters

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/recordsets"
	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/zones"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

type DesignateExporter struct {
	BaseOpenStackExporter
}

var zone_status = []string{
	"pending",
	"active",
	"deleted",
	"error",
}

var recordset_status = []string{
	"pending",
	"active",
	"deleted",
	"error",
}

func mapZoneStatus(zoneStatus string) int {
	for idx, status := range zone_status {
		if status == strings.ToLower(zoneStatus) {
			return idx
		}
	}
	return -1
}

func mapRecordsetStatus(recordsetStatus string) int {
	for idx, status := range recordset_status {
		if status == strings.ToLower(recordsetStatus) {
			return idx
		}
	}
	return -1
}

var defaultDesignateMetrics = []Metric{
	{Name: "zones", Fn: ListZonesAndRecordsets},
	{Name: "agent_state", Labels: []string{"id", "hostname", "service", "status"}, Fn: ListDesignateServices},
	{Name: "zone_status", Labels: []string{"id", "name", "status", "tenant_id", "type"}, Fn: nil},
	{Name: "recordsets", Labels: []string{"zone_id", "zone_name", "tenant_id"}, Fn: nil},
	{Name: "recordsets_status", Labels: []string{"id", "name", "status", "zone_id", "zone_name", "type"}, Fn: nil},
}

func NewDesignateExporter(config *ExporterConfig, logger *slog.Logger) (*DesignateExporter, error) {
	exporter := DesignateExporter{
		BaseOpenStackExporter{
			ExporterConfig: *config,
			Name:           "designate",
			logger:         logger,
		},
	}

	// This header needed for colletiong zone of all projects
	exporter.ClientV2.MoreHeaders = map[string]string{"X-Auth-All-Projects": "True"}

	for _, metric := range defaultDesignateMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}

	return &exporter, nil
}

func ListZonesAndRecordsets(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	allPagesZones, err := zones.List(exporter.ClientV2, zones.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	allZones, err := zones.ExtractZones(allPagesZones)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["zones"].Metric,
		prometheus.GaugeValue, float64(len(allZones)))

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(exporter.GetDnsConcurrencyCount())

	// Collect recordsets for zone and write metrics for zones and recordsets
	for _, zone := range allZones {
		zone := zone
		g.Go(func() error {
			allPagesRecordsets, err := recordsets.ListByZone(exporter.ClientV2, zone.ID, recordsets.ListOpts{}).AllPages(gCtx)
			if err != nil {
				return err
			}

			allRecordsets, err := recordsets.ExtractRecordSets(allPagesRecordsets)
			if err != nil {
				return err
			}

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["recordsets"].Metric,
				prometheus.GaugeValue, float64(len(allRecordsets)), zone.ID, zone.Name, zone.ProjectID)

			for _, recordset := range allRecordsets {
				ch <- prometheus.MustNewConstMetric(exporter.Metrics["recordsets_status"].Metric,
					prometheus.GaugeValue, float64(mapRecordsetStatus(recordset.Status)), recordset.ID, recordset.Name,
					recordset.Status, recordset.ZoneID, recordset.ZoneName, recordset.Type)
			}

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["zone_status"].Metric,
				prometheus.GaugeValue, float64(mapZoneStatus(zone.Status)), zone.ID, zone.Name,
				zone.Status, zone.ProjectID, zone.Type)

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

type designateServiceStatus struct {
	ID          string `json:"id"`
	Hostname    string `json:"hostname"`
	ServiceName string `json:"service_name"`
	Status      string `json:"status"`
}

func ListDesignateServices(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var response struct {
		Services []designateServiceStatus `json:"service_statuses"`
	}
	if _, err := exporter.ClientV2.Get(ctx, exporter.ClientV2.ServiceURL("service_statuses"), &response, nil); err != nil {
		return err
	}

	for _, service := range response.Services {
		state := 0.0
		if strings.EqualFold(service.Status, "up") {
			state = 1
		}
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["agent_state"].Metric,
			prometheus.GaugeValue, state, service.ID, service.Hostname, service.ServiceName, service.Status)
	}

	return nil
}
