package exporters

import (
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/services"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/volumetenants"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type CinderExporter struct {
	BaseOpenStackExporter
}

var volume_status = []string{
	"creating",
	"available",
	"reserved",
	"attaching",
	"detaching",
	"in-use",
	"maintenance",
	"deleting",
	"awaiting-transfer",
	"error",
	"error_deleting",
	"backing-up",
	"restoring-backup",
	"error_backing-up",
	"error_restoring",
	"error_extending",
	"downloading",
	"uploading",
	"retyping",
	"extending",
}

func mapVolumeStatus(volStatus string) int {
	for idx, status := range volume_status {
		if status == strings.ToLower(volStatus) {
			return idx
		}
	}
	return -1
}

var defaultCinderMetrics = []Metric{
	{Name: "volumes", Fn: ListVolumes},
	{Name: "snapshots", Fn: ListSnapshots},
	{Name: "agent_state", Labels: []string{"hostname", "service", "adminState", "zone"}, Fn: ListCinderAgentState},
	{Name: "volume_status", Labels: []string{"id", "name", "status", "bootable", "tenant_id", "size", "volume_type"}, Fn: nil},
}

func NewCinderExporter(client *gophercloud.ServiceClient, prefix string, disabledMetrics []string) (*CinderExporter, error) {
	exporter := CinderExporter{
		BaseOpenStackExporter{
			Name:            "cinder",
			Prefix:          prefix,
			Client:          client,
			DisabledMetrics: disabledMetrics,
		},
	}
	for _, metric := range defaultCinderMetrics {
		exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, nil)
	}

	return &exporter, nil
}

func (exporter *CinderExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range exporter.Metrics {
		ch <- metric.Metric
	}
}

func ListVolumes(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {
	log.Infoln("Fetching volumes info")

	type VolumeWithExt struct {
		volumes.Volume
		volumetenants.VolumeTenantExt
	}

	var allVolumes []VolumeWithExt

	allPagesVolumes, err := volumes.List(exporter.Client, volumes.ListOpts{
		AllTenants: true,
	}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	err = volumes.ExtractVolumesInto(allPagesVolumes, &allVolumes)
	if err != nil {
		log.Errorln(err)
		return
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["volumes"].Metric,
		prometheus.GaugeValue, float64(len(allVolumes)))

	// Volume status metrics
	for _, volume := range allVolumes {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["volume_status"].Metric,
			prometheus.GaugeValue, float64(mapVolumeStatus(volume.Status)), volume.ID, volume.Name,
			volume.Status, volume.Bootable, volume.TenantID, strconv.Itoa(volume.Size), volume.VolumeType)
	}
}

func ListSnapshots(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching snapshots information")
	var allSnapshots []snapshots.Snapshot

	allPagesSnapshot, err := snapshots.List(exporter.Client, snapshots.ListOpts{AllTenants: true}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allSnapshots, err = snapshots.ExtractSnapshots(allPagesSnapshot)
	if err != nil {
		log.Errorln(err)
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["snapshots"].Metric,
		prometheus.GaugeValue, float64(len(allSnapshots)))
}

func ListCinderAgentState(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching services state information")
	var allServices []services.Service

	allPagesService, err := services.List(exporter.Client, services.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}
	allServices, err = services.ExtractServices(allPagesService)
	if err != nil {
		log.Errorln(err)
		return
	}

	for _, service := range allServices {
		var state int = 0
		if service.State == "up" {
			state = 1
		}
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["agent_state"].Metric,
			prometheus.CounterValue, float64(state), service.Host, service.Binary, service.Status, service.Zone)
	}
}

func (exporter *CinderExporter) Collect(ch chan<- prometheus.Metric) {
	exporter.CollectMetrics(ch)
}
