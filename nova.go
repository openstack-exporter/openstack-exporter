package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/services"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/hypervisors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"

)

type NovaExporter struct {
	BaseOpenStackExporter
	Client *gophercloud.ServiceClient
}

var defaultNovaMetrics = []Metric{
	{Name: "flavors"},
	{Name: "availability_zones"},
	{Name: "security_groups"},
	{Name: "total_vms"},
	{Name: "running_vms", Labels: []string{"hostname", "aggregate"}},
	{Name: "current_workload", Labels: []string{"hostname", "aggregate"}},
	{Name: "vcpus_available", Labels: []string{"hostname", "aggregate"}},
	{Name: "vcpus_used", Labels: []string{"hostname", "aggregate"}},
	{Name: "memory_available_bytes", Labels: []string{"hostname", "aggregate"}},
	{Name: "memory_used_bytes", Labels: []string{"hostname", "aggregate"}},
	{Name: "local_storage_available_bytes", Labels: []string{"hostname", "aggregate"}},
	{Name: "local_storage_used_bytes", Labels: []string{"hostname", "aggregate"}},
	{Name: "agent_state", Labels: []string{"hostname", "service", "adminState", "zone"}},
}

func NewNovaExporter(client *gophercloud.ProviderClient, prefix string, config *Cloud) (*NovaExporter, error) {
	compute, err := openstack.NewComputeV2(client, gophercloud.EndpointOpts{
		Region:       "RegionOne",
		Availability: "internal",
	})
	if err != nil {
		log.Errorln(err)
	}

	exporter := NovaExporter{
		BaseOpenStackExporter{
			Name:                 "nova",
			Prefix:               prefix,
			Config:               config,
			AuthenticatedClient: client,
		}, compute,
	}

	for _, metric := range defaultNovaMetrics {
		exporter.AddMetric(metric.Name, metric.Labels, nil)
	}

	return &exporter, nil
}

func (exporter *NovaExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range exporter.Metrics {
		ch <- metric
	}
}

func (exporter *NovaExporter) RefreshClient() error {
	log.Infoln("Refresh auth client, in case token has expired")
	client, err := openstack.NewComputeV2(exporter.AuthenticatedClient, gophercloud.EndpointOpts{
		Region:       "RegionOne",
		Availability: "internal",
	})
	if err != nil {
		log.Errorln(err)
	}
	exporter.Client = client
	return nil
}

func (exporter *NovaExporter) Collect(ch chan<- prometheus.Metric) {
	log.Infoln("Fetching list of services")
	var allServices []services.Service

	allPagesServices, err := services.List(exporter.Client).AllPages()
	allServices, err = services.ExtractServices(allPagesServices)
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

	log.Infoln("Fetching list of hypervisors")
	var allHypervisors []hypervisors.Hypervisor

	allPagesHypervisors, err := hypervisors.List(exporter.Client).AllPages()
	allHypervisors, err = hypervisors.ExtractHypervisors(allPagesHypervisors)
	if err != nil {
		log.Errorln(err)
	}

	for _, hypervisor := range allHypervisors {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["running_vms"],
			prometheus.GaugeValue, float64(hypervisor.RunningVMs), hypervisor.HypervisorHostname, "")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["current_workload"],
			prometheus.GaugeValue, float64(hypervisor.CurrentWorkload), hypervisor.HypervisorHostname, "")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["vcpus_available"],
			prometheus.GaugeValue, float64(hypervisor.VCPUs), hypervisor.HypervisorHostname, "")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["vcpus_used"],
			prometheus.GaugeValue, float64(hypervisor.VCPUsUsed), hypervisor.HypervisorHostname, "")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["memory_available_bytes"],
			prometheus.GaugeValue, float64(hypervisor.MemoryMB*MEGABYTE), hypervisor.HypervisorHostname, "")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["memory_used_bytes"],
			prometheus.GaugeValue, float64(hypervisor.MemoryMBUsed*MEGABYTE), hypervisor.HypervisorHostname, "")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["local_storage_available_bytes"],
			prometheus.GaugeValue, float64(hypervisor.LocalGB*GIGABYTE), hypervisor.HypervisorHostname, "")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["local_storage_used_bytes"],
			prometheus.GaugeValue, float64(hypervisor.LocalGBUsed*GIGABYTE), hypervisor.HypervisorHostname, "")
	}

	log.Infoln("Fetching list of flavors")
	var allFlavors []flavors.Flavor

	allPagesFlavors, err := flavors.ListDetail(exporter.Client, flavors.ListOpts{}).AllPages()
	allFlavors, err = flavors.ExtractFlavors(allPagesFlavors)
	if err != nil {
		log.Errorln(err)
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["flavors"],
		prometheus.GaugeValue, float64(len(allFlavors)))

	log.Infoln("Fetching list of availability zones")
	var allAZs []availabilityzones.AvailabilityZone

	allPagesAZs, err := availabilityzones.List(exporter.Client).AllPages()
	allAZs, err = availabilityzones.ExtractAvailabilityZones(allPagesAZs)
	if err != nil {
		log.Errorln(err)
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["availability_zones"],
		prometheus.GaugeValue, float64(len(allAZs)))
}