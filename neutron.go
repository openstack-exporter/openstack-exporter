package main

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type NeutronExporter struct {
	BaseOpenStackExporter
}

var defaultNeutronMetrics = []Metric{
	{Name: "floating_ips"},
	{Name: "networks"},
	{Name: "security_groups"},
	{Name: "subnets"},
	{Name: "agent_state", Labels: []string{"hostname", "service", "adminState"}},
}

func NewNeutronExporter(client *gophercloud.ServiceClient, prefix string) (*NeutronExporter, error) {
	exporter := NeutronExporter{
		BaseOpenStackExporter{
			Name:   "neutron",
			Prefix: prefix,
			Client: client,
		},
	}

	for _, metric := range defaultNeutronMetrics {
		exporter.AddMetric(metric.Name, metric.Labels, nil)
	}

	return &exporter, nil
}

func (exporter *NeutronExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range exporter.Metrics {
		ch <- metric
	}
}

func (exporter *NeutronExporter) RefreshClient() error {
	log.Infoln("Refreshing auth client in case token has expired")
	return nil
}

func (exporter *NeutronExporter) Collect(ch chan<- prometheus.Metric) {
	if err := exporter.RefreshClient(); err != nil {
		log.Error(err)
		return
	}

	log.Infoln("Fetching floating ips list")
	allPagesFloatingIPs, _ := floatingips.List(exporter.Client, floatingips.ListOpts{}).AllPages()
	fmt.Println(allPagesFloatingIPs)
	//if err != nil {
	//	log.Errorf("%s", err)
	//}
	//
	//log.Infoln("Fetching agents list")
	//agents, err := exporter.Client.ListAgentsV2()
	//if err != nil {
	//	log.Errorf(err.Error())
	//}
	//
	//for _, agent := range agents {
	//	var state int = 0
	//	if agent.Alive {
	//		state = 1
	//	}
	//
	//	adminState := "down"
	//	if agent.AdminStateUp {
	//		adminState = "up"
	//	}
	//	ch <- prometheus.MustNewConstMetric(exporter.Metrics["agent_state"],
	//		prometheus.CounterValue, float64(state), agent.Host, agent.Binary, adminState)
	//}
	//
	//log.Infoln("Fetching list of networks")
	//networks, err := exporter.Client.ListNetworksV2()
	//if err != nil {
	//	log.Errorf("%s", err)
	//}
	//
	//log.Infoln("Fetching list of security groups")
	//securityGroups, err := exporter.Client.ListSecurityGroupsV2()
	//if err != nil {
	//	log.Errorf("%s", err)
	//}
	//
	//log.Infoln("Fetching list of subnets")
	//subnets, err := exporter.Client.ListSubnetsV2()
	//if err != nil {
	//	log.Errorf("%s", err)
	//}
	//
	//ch <- prometheus.MustNewConstMetric(exporter.Metrics["subnets"],
	//	prometheus.GaugeValue, float64(len(subnets)))
	//
	//ch <- prometheus.MustNewConstMetric(exporter.Metrics["floating_ips"],
	//	prometheus.GaugeValue, float64(len(floatingips)))
	//
	//ch <- prometheus.MustNewConstMetric(exporter.Metrics["networks"],
	//	prometheus.GaugeValue, float64(len(networks)))
	//
	//ch <- prometheus.MustNewConstMetric(exporter.Metrics["security_groups"],
	//	prometheus.GaugeValue, float64(len(securityGroups)))
}