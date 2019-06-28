package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"gopkg.in/niedbalski/goose.v3/cinder"
	"gopkg.in/niedbalski/goose.v3/client"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type CinderExporter struct {
	BaseOpenStackExporter
	Client *cinder.Client
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

func NewCinderExporter(client client.AuthenticatingClient, prefix string, config *Cloud) (*CinderExporter, error) {
	endpoint := client.EndpointsForRegion(config.Region)["volumev3"]
	endpointUrl, err := url.Parse(endpoint)

	if err != nil {
		return nil, err
	}

	exporter := CinderExporter{BaseOpenStackExporter{
		Name:   "cinder",
		Prefix: prefix,
		Config: config,
	}, cinder.NewClient(client.TenantId(), endpointUrl,
		cinder.SetAuthHeaderFn(client.Token,
			http.DefaultClient.Do),
	)}

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

func (exporter *CinderExporter) Collect(ch chan<- prometheus.Metric) {
	log.Infoln("Fetching volumes info")
	volumes, err := exporter.Client.GetVolumesSimple(true)
	if err != nil {
		log.Errorf("%s", err)
		return
	}

	log.Infoln("Fetching volumes information")
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["volumes"],
		prometheus.GaugeValue, float64(len(volumes.Volumes)))

	// Server status metrics
	for _, volume := range volumes.Volumes {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["volume_status"],
			prometheus.GaugeValue, float64(mapVolumeStatus(volume.Status)), volume.ID, volume.Name,
			volume.Status, volume.Bootable, volume.Os_Vol_Tenant_Attr_TenantID, strconv.Itoa(volume.Size), volume.VolumeType)
	}

	log.Infoln("Fetching snapshots information")
	snapshots, err := exporter.Client.GetSnapshotsSimple(true)
	if err != nil {
		log.Errorf("%s", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["snapshots"],
		prometheus.GaugeValue, float64(len(snapshots.Snapshots)))

	log.Infoln("Fetching services state information")
	services, err := exporter.Client.GetServices()
	if err != nil {
		log.Errorf("%s", err)
		return
	}

	for _, service := range services.Services {
		var state int = 0
		if service.State == "up" {
			state = 1
		}
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["agent_state"],
			prometheus.CounterValue, float64(state), service.Host, service.Binary, service.Status, service.Zone)
	}

}
