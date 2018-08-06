package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"gopkg.in/niedbalski/goose.v3/client"
	"gopkg.in/niedbalski/goose.v3/neutron"
)

type NeutronExporter struct {
	BaseOpenStackExporter
	Client *neutron.Client
}

var defaultNeutronMetrics = []Metric{
	{Name: "floating_ips"},
	{Name: "networks"},
	{Name: "security_groups"},
	{Name: "subnets"},
	{Name: "agent_state", Labels: []string{"hostname", "service", "adminState"}},
}

func NewNeutronExporter(client client.AuthenticatingClient, config *Cloud) (*NeutronExporter, error) {
	exporter := NeutronExporter{
		BaseOpenStackExporter{Name: "neutron", Config: config},
		neutron.New(client),
	}

	for _, metric := range defaultNeutronMetrics {
		exporter.AddMetric(metric.Name, metric.Labels, nil)
	}

	return &exporter, nil
}

func (exporter *NeutronExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range exporter.GetMetrics() {
		ch <- metric
	}
}

func (exporter *NeutronExporter) Collect(ch chan<- prometheus.Metric) {
	floatingips, err := exporter.Client.ListFloatingIPsV2()
	if err != nil {
		log.Errorf("%s", err)
	}

	agents, err := exporter.Client.ListAgentsV2()
	if err != nil {
		log.Errorf(err.Error())
	}

	for _, agent := range agents {
		var state int = 0
		if agent.Alive == true {
			state = 1
		}

		adminState := "down"
		if agent.AdminStateUp == true {
			adminState = "up"
		}
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["agent_state"],
			prometheus.CounterValue, float64(state), agent.Host, agent.Binary, adminState)
	}

	networks, err := exporter.Client.ListNetworksV2()
	if err != nil {
		log.Errorf("%s", err)
	}

	securityGroups, err := exporter.Client.ListSecurityGroupsV2()
	if err != nil {
		log.Errorf("%s", err)
	}

	subnets, err := exporter.Client.ListSubnetsV2()
	if err != nil {
		log.Errorf("%s", err)
	}

	ch <- prometheus.MustNewConstMetric(exporter.GetMetrics()["subnets"],
		prometheus.GaugeValue, float64(len(subnets)))

	ch <- prometheus.MustNewConstMetric(exporter.GetMetrics()["floating_ips"],
		prometheus.GaugeValue, float64(len(floatingips)))

	ch <- prometheus.MustNewConstMetric(exporter.GetMetrics()["networks"],
		prometheus.GaugeValue, float64(len(networks)))

	ch <- prometheus.MustNewConstMetric(exporter.GetMetrics()["security_groups"],
		prometheus.GaugeValue, float64(len(securityGroups)))
}
