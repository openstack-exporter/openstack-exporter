package exporters

import (
	"log/slog"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/dns/v2/recordsets"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/zones"
	"github.com/prometheus/client_golang/prometheus"
)

const DESIGNATE_SERVICE string = "designate"

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
			labels := computeMetricLabels(DESIGNATE_SERVICE, metric, exporter.ExtraLabels)
			constLabels := computeConstantLabels(DESIGNATE_SERVICE, metric, exporter.ExtraLabels)
			exporter.AddMetric(metric.Name, metric.Fn, labels, metric.DeprecatedVersion, constLabels)
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

	zoneStatusSpec := exporter.ExtraLabels.Extract(DESIGNATE_SERVICE, "zone_status")
	recordsetsSpec := exporter.ExtraLabels.Extract(DESIGNATE_SERVICE, "recordsets")
	recordsetsStatusSpec := exporter.ExtraLabels.Extract(DESIGNATE_SERVICE, "recordsets_status")
	// Collect recordsets for zone and write metrics for zones and recordsets
	for _, zone := range allZones {

		allPagesRecordsets, err := recordsets.ListByZone(exporter.Client, zone.ID, recordsets.ListOpts{}).AllPages()
		if err != nil {
			return err
		}

		allRecordsets, err := recordsets.ExtractRecordSets(allPagesRecordsets)
		if err != nil {
			return err
		}

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["recordsets"].Metric,
			prometheus.GaugeValue, float64(len(allRecordsets)), append([]string{zone.ID, zone.Name, zone.ProjectID}, resolveExtraLabelValues(zone, recordsetsSpec)...)...)

		for _, recordset := range allRecordsets {
			extraValues := resolveExtraLabelValues(recordset, recordsetsStatusSpec)
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["recordsets_status"].Metric,
				prometheus.GaugeValue, float64(mapRecordsetStatus(recordset.Status)), append([]string{recordset.ID, recordset.Name,
					recordset.Status, recordset.ZoneID, recordset.ZoneName, recordset.Type}, extraValues...)...)
		}

		extraValues := resolveExtraLabelValues(zone, zoneStatusSpec)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["zone_status"].Metric,
			prometheus.GaugeValue, float64(mapZoneStatus(zone.Status)), append([]string{zone.ID, zone.Name,
				zone.Status, zone.ProjectID, zone.Type}, extraValues...)...)

	}

	return nil
}
