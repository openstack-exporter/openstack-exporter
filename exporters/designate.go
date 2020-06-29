package exporters

import (
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/recordsets"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/zones"
	"github.com/prometheus/client_golang/prometheus"
)

type DesignateExporter struct {
	BaseOpenStackExporter
}

// var volume_status = []string{
// 	"creating",
// 	"available",
// 	"reserved",
// 	"attaching",
// 	"detaching",
// 	"in-use",
// 	"maintenance",
// 	"deleting",
// 	"awaiting-transfer",
// 	"error",
// 	"error_deleting",
// 	"backing-up",
// 	"restoring-backup",
// 	"error_backing-up",
// 	"error_restoring",
// 	"error_extending",
// 	"downloading",
// 	"uploading",
// 	"retyping",
// 	"extending",
// }

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
	{Name: "recordsets", Fn: nil},
	{Name: "recordsets_status", Labels: []string{"id", "name", "status", "zone_id", "zone_name", "type"}, Fn: nil},
}

func NewDesignateExporter(client *gophercloud.ServiceClient, prefix string, disabledMetrics []string) (*DesignateExporter, error) {
	exporter := DesignateExporter{
		BaseOpenStackExporter{
			Name:            "designate",
			Prefix:          prefix,
			Client:          client,
			DisabledMetrics: disabledMetrics,
		},
	}
	for _, metric := range defaultDesignateMetrics {
		exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, nil)
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
			prometheus.GaugeValue, float64(len(allRecordsets)))

		for _, recordset := range allRecordsets {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["recordsets_status"].Metric,
				prometheus.GaugeValue, float64(mapRecordsetStatus(recordset.Status)), recordset.ID, recordset.Name,
				recordset.Status, recordset.ZoneID, recordset.ZoneName, recordset.Type)
		}

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["zone_status"].Metric,
			prometheus.GaugeValue, float64(mapZoneStatus(zone.Status)), zone.ID, zone.Name,
			zone.Status, zone.ProjectID, zone.Type)
	}

	return nil
}
