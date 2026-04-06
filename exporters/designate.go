package exporters

import (
	"log/slog"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/dns/v2/recordsets"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/zones"
	"github.com/prometheus/client_golang/prometheus"
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
	exporter.Client.MoreHeaders = map[string]string{"X-Auth-All-Projects": "True"}

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

func ListZonesAndRecordsets(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	allPagesZones, err := zones.List(exporter.Client, zones.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allZones, err := zones.ExtractZones(allPagesZones)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["zones"].Metric,
		prometheus.GaugeValue, float64(len(allZones)))

	// Fetch all recordsets in one go (Designate API supports listing across all zones)
	allPagesRecordsets, err := recordsets.ListRecordSets(exporter.Client, "all", recordsets.ListOpts{Limit: 1000}).AllPages()
	if err != nil {
		return err
	}

	allRecordsets, err := recordsets.ExtractRecordSets(allPagesRecordsets)
	if err != nil {
		return err
	}

	zoneCounts := make(map[string]int)

	for _, recordset := range allRecordsets {
		zoneCounts[recordset.ZoneName] = zoneCounts[recordset.ZoneName] + 1
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["recordsets_status"].Metric,
			prometheus.GaugeValue, float64(mapRecordsetStatus(recordset.Status)), recordset.ID, recordset.Name,
			recordset.Status, recordset.ZoneID, recordset.ZoneName, recordset.Type)
	}

	// Emit zone related metrics
	for _, zone := range allZones {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["recordsets"].Metric,
			prometheus.GaugeValue, float64(zoneCounts[zone.Name]), zone.ID, zone.Name, zone.ProjectID)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["zone_status"].Metric,
			prometheus.GaugeValue, float64(mapZoneStatus(zone.Status)), zone.ID, zone.Name,
			zone.Status, zone.ProjectID, zone.Type)
	}

	return nil
}
