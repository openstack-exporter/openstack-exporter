package exporters

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/agents"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/networkipavailabilities"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"strconv"
)

type NeutronExporter struct {
	BaseOpenStackExporter
}

var defaultNeutronMetrics = []Metric{
	{Name: "floating_ips", Fn: ListFloatingIps},
	{Name: "networks", Fn: ListNetworks},
	{Name: "security_groups", Fn: ListSecGroups},
	{Name: "subnets", Fn: ListSubnets},
	{Name: "ports", Fn: ListPorts},
	{Name: "routers", Fn: ListRouters},
	{Name: "agent_state", Labels: []string{"hostname", "service", "adminState"}, Fn: ListAgentStates},
	{Name: "network_ip_availabilities_total", Labels: []string{"network_id", "network_name", "cidr", "subnet_name", "project_id"}, Fn: ListNetworkIPAvailabilities},
	{Name: "network_ip_availabilities_used", Labels: []string{"network_id", "network_name", "cidr", "subnet_name", "project_id"}},
}

func NewNeutronExporter(client *gophercloud.ServiceClient, prefix string, disabledMetrics []string) (*NeutronExporter, error) {
	exporter := NeutronExporter{
		BaseOpenStackExporter{
			Name:            "neutron",
			Prefix:          prefix,
			Client:          client,
			DisabledMetrics: disabledMetrics,
		},
	}

	for _, metric := range defaultNeutronMetrics {
		exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, nil)
	}

	return &exporter, nil
}

func (exporter *NeutronExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range exporter.Metrics {
		ch <- metric.Metric
	}
}

func (exporter *NeutronExporter) Collect(ch chan<- prometheus.Metric) {
	exporter.CollectMetrics(ch)
}

func ListFloatingIps(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching floating ips list")
	var allFloatingIPs []floatingips.FloatingIP

	allPagesFloatingIPs, err := floatingips.List(exporter.Client, floatingips.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allFloatingIPs, err = floatingips.ExtractFloatingIPs(allPagesFloatingIPs)
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["floating_ips"].Metric,
		prometheus.GaugeValue, float64(len(allFloatingIPs)))
}

func ListAgentStates(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching agents list")
	var allAgents []agents.Agent

	allPagesAgents, err := agents.List(exporter.Client, agents.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allAgents, err = agents.ExtractAgents(allPagesAgents)
	if err != nil {
		log.Errorln(err)
		return
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
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["agent_state"].Metric,
			prometheus.CounterValue, float64(state), agent.Host, agent.Binary, adminState)
	}
}

func ListNetworks(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching list of networks")
	var allNetworks []networks.Network

	allPagesNetworks, err := networks.List(exporter.Client, networks.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allNetworks, err = networks.ExtractNetworks(allPagesNetworks)
	if err != nil {
		log.Errorln(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["networks"].Metric,
		prometheus.GaugeValue, float64(len(allNetworks)))
}

func ListSecGroups(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching list of network security groups")
	var allSecurityGroups []groups.SecGroup

	allPagesSecurityGroups, err := groups.List(exporter.Client, groups.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allSecurityGroups, err = groups.ExtractGroups(allPagesSecurityGroups)
	if err != nil {
		log.Errorln(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["security_groups"].Metric,
		prometheus.GaugeValue, float64(len(allSecurityGroups)))
}

func ListSubnets(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching list of subnets")
	var allSubnets []subnets.Subnet

	allPagesSubnets, err := subnets.List(exporter.Client, subnets.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allSubnets, err = subnets.ExtractSubnets(allPagesSubnets)
	if err != nil {
		log.Errorln(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["subnets"].Metric,
		prometheus.GaugeValue, float64(len(allSubnets)))
}

func ListPorts(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching list of ports")
	var allPorts []ports.Port

	allPagesPorts, err := ports.List(exporter.Client, ports.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allPorts, err = ports.ExtractPorts(allPagesPorts)
	if err != nil {
		log.Errorln(err)
		return
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["ports"].Metric,
		prometheus.GaugeValue, float64(len(allPorts)))
}

func ListNetworkIPAvailabilities(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching network ip availabilities list")
	var allNetworkIPAvailabilities []networkipavailabilities.NetworkIPAvailability

	allPagesNetworkIPAvailabilities, err := networkipavailabilities.List(exporter.Client, networkipavailabilities.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allNetworkIPAvailabilities, err = networkipavailabilities.ExtractNetworkIPAvailabilities(allPagesNetworkIPAvailabilities)
	if err != nil {
		log.Errorln(err)
	}

	for _, NetworkIPAvailabilities := range allNetworkIPAvailabilities {
		for _, SubnetIPAvailability := range NetworkIPAvailabilities.SubnetIPAvailabilities {
			totalIPs, err := strconv.ParseFloat(SubnetIPAvailability.TotalIPs, 64)
			if err != nil {
				log.Errorln(err)
				return
			}
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["network_ip_availabilities_total"].Metric,
				prometheus.GaugeValue, totalIPs, NetworkIPAvailabilities.NetworkID,
				NetworkIPAvailabilities.NetworkName, SubnetIPAvailability.CIDR,
				SubnetIPAvailability.SubnetName, NetworkIPAvailabilities.ProjectID)

			usedIPs, err := strconv.ParseFloat(SubnetIPAvailability.UsedIPs, 64)
			if err != nil {
				log.Errorln(err)
				return
			}
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["network_ip_availabilities_used"].Metric,
				prometheus.GaugeValue, usedIPs, NetworkIPAvailabilities.NetworkID,
				NetworkIPAvailabilities.NetworkName, SubnetIPAvailability.CIDR,
				SubnetIPAvailability.SubnetName, NetworkIPAvailabilities.ProjectID)
		}
	}
}

func ListRouters(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching list of routers")
	var allRouters []routers.Router

	allPagesRouters, err := routers.List(exporter.Client, routers.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allRouters, err = routers.ExtractRouters(allPagesRouters)
	if err != nil {
		log.Errorln(err)
		return
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["routers"].Metric,
		prometheus.GaugeValue, float64(len(allRouters)))
}
