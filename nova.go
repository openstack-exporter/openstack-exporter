package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"gopkg.in/niedbalski/goose.v3/client"
	"gopkg.in/niedbalski/goose.v3/nova"
)

type NovaExporter struct {
	BaseOpenStackExporter
	Client *nova.Client
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

func NewNovaExporter(client client.AuthenticatingClient, prefix string, config *Cloud) (*NovaExporter, error) {
	exporter := NovaExporter{
		BaseOpenStackExporter{
			Name:   "nova",
			Prefix: prefix,
			Config: config,
		}, nova.New(client)}

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

func (exporter *NovaExporter) Collect(ch chan<- prometheus.Metric) {
	log.Infoln("Fetching list of services")
	services, err := exporter.Client.ListServices()
	if err != nil {
		log.Errorf(err.Error())
	}

	for _, service := range services {
		var state int = 0
		if service.State == "up" {
			state = 1
		}
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["agent_state"],
			prometheus.CounterValue, float64(state), service.Host, service.Binary, service.Status, service.Zone)
	}

	log.Infoln("Fetching list of hypervisors")
	hypervisors, err := exporter.Client.ListHypervisors()
	if err != nil {
		log.Errorf("%v", err)
	}

	for _, hypervisor := range hypervisors {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["running_vms"],
			prometheus.GaugeValue, float64(hypervisor.RunningVms), hypervisor.HypervisorHostname, "")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["current_workload"],
			prometheus.GaugeValue, float64(hypervisor.CurrentWorkload), hypervisor.HypervisorHostname, "")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["vcpus_available"],
			prometheus.GaugeValue, float64(hypervisor.Vcpus), hypervisor.HypervisorHostname, "")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["vcpus_used"],
			prometheus.GaugeValue, float64(hypervisor.VcpusUsed), hypervisor.HypervisorHostname, "")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["memory_available_bytes"],
			prometheus.GaugeValue, float64(hypervisor.MemoryMb*MEGABYTE), hypervisor.HypervisorHostname, "")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["memory_used_bytes"],
			prometheus.GaugeValue, float64(hypervisor.MemoryMbUsed*MEGABYTE), hypervisor.HypervisorHostname, "")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["local_storage_available_bytes"],
			prometheus.GaugeValue, float64(hypervisor.LocalGb*GIGABYTE), hypervisor.HypervisorHostname, "")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["local_storage_used_bytes"],
			prometheus.GaugeValue, float64(hypervisor.LocalGbUsed*GIGABYTE), hypervisor.HypervisorHostname, "")
	}

	log.Infoln("Fetching list of flavors")
	flavors, err := exporter.Client.ListFlavors()
	if err != nil {
		log.Errorf("%s", err)
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["flavors"],
		prometheus.GaugeValue, float64(len(flavors)))

	log.Infoln("Fetching list of availability zones")
	azs, err := exporter.Client.ListAvailabilityZones()
	if err != nil {
		log.Errorf("%s", err)
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["availability_zones"],
		prometheus.GaugeValue, float64(len(azs)))

	log.Infoln("Fetching list of security groups")
	securtyGroups, err := exporter.Client.ListSecurityGroups()
	if err != nil {
		log.Errorf("%s", err)
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["security_groups"],
		prometheus.GaugeValue, float64(len(securtyGroups)))

	filter := nova.NewFilter()
	filter.Set("all_tenants", "1")
	log.Infoln("Fetching list of instances for all tenants")
	servers, err := exporter.Client.ListServers(filter)
	if err != nil {
		log.Errorf("%s", err)
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["total_vms"],
		prometheus.GaugeValue, float64(len(servers)))
}
