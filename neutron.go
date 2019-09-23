package main

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/agents"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type NeutronExporter struct {
	BaseOpenStackExporter
	Client *gophercloud.ServiceClient
}

var defaultNeutronMetrics = []Metric{
	{Name: "floating_ips"},
	{Name: "networks"},
	{Name: "security_groups"},
	{Name: "subnets"},
	{Name: "ports"},
	{Name: "agent_state", Labels: []string{"hostname", "service", "adminState"}},
}

func NewNeutronExporter(client *gophercloud.ProviderClient, prefix string, config *Cloud) (*NeutronExporter, error) {
	network, err := openstack.NewNetworkV2(client, gophercloud.EndpointOpts{
		Region:       "RegionOne",
		Availability: "internal",
	})
	if err != nil {
		log.Errorln(err)
	}

	exporter := NeutronExporter{
		BaseOpenStackExporter{
			Name:                 "neutron",
			Prefix:               prefix,
			Config:               config,
			AuthenticatedClient: client,
		},network,
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
	log.Infoln("Refresh auth client, in case token has expired")
	client, err := openstack.NewNetworkV2(exporter.AuthenticatedClient, gophercloud.EndpointOpts{
		Region:       "RegionOne",
		Availability: "internal",
	})
	if err != nil {
		log.Errorln(err)
	}
	exporter.Client = client
	return nil
}

func (exporter *NeutronExporter) Collect(ch chan<- prometheus.Metric) {
	log.Infoln("Fetching floating ips list")
	var allFloatingIPs []floatingips.FloatingIP

	allPagesFloatingIPs, err := floatingips.List(exporter.Client, floatingips.ListOpts{}).AllPages()
	allFloatingIPs, err = floatingips.ExtractFloatingIPs(allPagesFloatingIPs)
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["floating_ips"],
		prometheus.GaugeValue, float64(len(allFloatingIPs)))

	log.Infoln("Fetching agents list")
	var allAgents []agents.Agent

	allPagesAgents, err := agents.List(exporter.Client, agents.ListOpts{}).AllPages()
	allAgents, err = agents.ExtractAgents(allPagesAgents)
	if err != nil {
		log.Errorln(err)
	}

	for _, agent := range allAgents {
		var state int = 0
		if agent.Alive {
			state = 1
		}

		adminState := "down"
		if agent.AdminStateUp {
			adminState = "up"
		}
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["agent_state"],
			prometheus.CounterValue, float64(state), agent.Host, agent.Binary, adminState)
	}

	log.Infoln("Fetching list of networks")
	var allNetworks []networks.Network

	allPagesNetworks, err := networks.List(exporter.Client, networks.ListOpts{}).AllPages()
	allNetworks, err = networks.ExtractNetworks(allPagesNetworks)
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["networks"],
		prometheus.GaugeValue, float64(len(allNetworks)))

	log.Infoln("Fetching list of security groups")
	var allSecurityGroups []groups.SecGroup

	allPagesSecurityGroups, err := groups.List(exporter.Client, groups.ListOpts{}).AllPages()
	allSecurityGroups, err = groups.ExtractGroups(allPagesSecurityGroups)
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["security_groups"],
		prometheus.GaugeValue, float64(len(allSecurityGroups)))

	log.Infoln("Fetching list of subnets")
	var allSubnets []subnets.Subnet

	allPagesSubnets, err := subnets.List(exporter.Client, subnets.ListOpts{}).AllPages()
	allSubnets, err = subnets.ExtractSubnets(allPagesSubnets)
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["subnets"],
		prometheus.GaugeValue, float64(len(allSubnets)))

	log.Infoln("Fetching list of ports")
	var allPorts []ports.Port

	allPagesPorts, err := ports.List(exporter.Client, ports.ListOpts{}).AllPages()
	allPorts, err = ports.ExtractPorts(allPagesPorts)
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["ports"],
		prometheus.GaugeValue, float64(len(allPorts)))

}