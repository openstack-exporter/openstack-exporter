package main

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/services"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"strconv"
	"strings"
)

type CinderExporter struct {
	BaseOpenStackExporter
	Client *gophercloud.ServiceClient
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
	{Name: "volumes"},
	{Name: "snapshots"},
	{Name: "agent_state", Labels: []string{"hostname", "service", "adminState", "zone"}},
	{Name: "volume_status", Labels: []string{"id", "name", "status", "bootable", "tenant_id", "size", "volume_type"}},
}

func NewCinderExporter(client *gophercloud.ProviderClient, prefix string, config *Cloud) (*CinderExporter, error) {
	block, err := openstack.NewBlockStorageV3(client, gophercloud.EndpointOpts{
		Region:       "RegionOne",
		Availability: "internal",
	})
	if err != nil {
		log.Errorln(err)
	}

	exporter := CinderExporter{BaseOpenStackExporter{
		Name:                 "cinder",
		Prefix:               prefix,
		Config:               config,
		AuthenticatedClient:  client,
		}, block,
	}

	for _, metric := range defaultCinderMetrics {
		exporter.AddMetric(metric.Name, metric.Labels, nil)
	}

	return &exporter, nil
}

func (exporter *CinderExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range exporter.Metrics {
		ch <- metric
	}
}

func (exporter *CinderExporter) RefreshClient() error {
	log.Infoln("Refresh auth client, in case token has expired")
	client, err := openstack.NewBlockStorageV3(exporter.AuthenticatedClient, gophercloud.EndpointOpts{
		Region:       "RegionOne",
		Availability: "internal",
	})
	if err != nil {
		log.Errorln(err)
	}
	exporter.Client = client
	return nil
}

func (exporter *CinderExporter) Collect(ch chan<- prometheus.Metric) {
	log.Infoln("Fetching volumes info")
	var allVolumes []volumes.Volume

	allPagesVolumes, err := volumes.List(exporter.Client,
		                                 volumes.ListOpts{
											 AllTenants: true,
										 }).AllPages()
	allVolumes, err = volumes.ExtractVolumes(allPagesVolumes)
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["volumes"],
		prometheus.GaugeValue, float64(len(allVolumes)))

	// Volume status metrics
	for _, volume := range allVolumes {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["volume_status"],
			prometheus.GaugeValue, float64(mapVolumeStatus(volume.Status)), volume.ID, volume.Name,
			volume.Status, volume.Bootable, "", strconv.Itoa(volume.Size), volume.VolumeType)
	}

	log.Infoln("Fetching snapshots information")
	var allSnapshots []snapshots.Snapshot

	allPagesSnapshot, err := snapshots.List(exporter.Client, snapshots.ListOpts{}).AllPages()
	allSnapshots, err = snapshots.ExtractSnapshots(allPagesSnapshot)
	if err != nil {
		log.Errorln(err)
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["snapshots"],
		prometheus.GaugeValue, float64(len(allSnapshots)))

	log.Infoln("Fetching services state information")
	var allServices []services.Service

	allPagesService, err := services.List(exporter.Client, services.ListOpts{}).AllPages()
	allServices, err = services.ExtractServices(allPagesService)
	if err != nil {
		log.Errorln(err)
	}

	for _, service := range allServices {
		var state int = 0
		if service.State == "up" {
			state = 1
		}
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["agent_state"],
			prometheus.CounterValue, float64(state), service.Host, service.Binary, service.Status, service.Zone)
	}

}