package exporters

import (
	"strconv"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/agents"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/loadbalancers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/networkipavailabilities"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/prometheus/client_golang/prometheus"
)

// NeutronExporter : extends BaseOpenStackExporter
type NeutronExporter struct {
	BaseOpenStackExporter
}

var defaultNeutronMetrics = []Metric{
	{Name: "floating_ips", Fn: ListFloatingIps},
	{Name: "floating_ips_associated_not_active"},
	{Name: "networks", Fn: ListNetworks},
	{Name: "security_groups", Fn: ListSecGroups},
	{Name: "subnets", Fn: ListSubnets},
	{Name: "port", Labels: []string{"uuid", "network_id", "mac_address", "device_owner", "status", "binding_vif_type"}, Fn: ListPorts},
	{Name: "ports"},
	{Name: "ports_no_ips"},
	{Name: "ports_lb_not_active"},
	{Name: "routers", Fn: ListRouters},
	{Name: "routers_not_active"},
	{Name: "agent_state", Labels: []string{"hostname", "service", "adminState"}, Fn: ListAgentStates},
	{Name: "network_ip_availabilities_total", Labels: []string{"network_id", "network_name", "ip_version", "cidr", "subnet_name", "project_id"}, Fn: ListNetworkIPAvailabilities},
	{Name: "network_ip_availabilities_used", Labels: []string{"network_id", "network_name", "ip_version", "cidr", "subnet_name", "project_id"}},
	{Name: "loadbalancers", Fn: ListLBs},
	{Name: "loadbalancers_not_active"},
}

// NewNeutronExporter : returns a pointer to NeutronExporter
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

// ListFloatingIps : count total number of instantiated FloatingIPs and those that are associated to private IP but not in ACTIVE state
func ListFloatingIps(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allFloatingIPs []floatingips.FloatingIP

	allPagesFloatingIPs, err := floatingips.List(exporter.Client, floatingips.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allFloatingIPs, err = floatingips.ExtractFloatingIPs(allPagesFloatingIPs)
	if err != nil {
		return err
	}

	failedFIPs := 0
	for _, fip := range allFloatingIPs {
		if fip.FixedIP != "" {
			if fip.Status != "ACTIVE" {
				failedFIPs = failedFIPs + 1
			}
		}
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["floating_ips"].Metric,
		prometheus.GaugeValue, float64(len(allFloatingIPs)))
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["floating_ips_associated_not_active"].Metric,
		prometheus.GaugeValue, float64(failedFIPs))

	return nil
}

// ListAgentStates : list agent state per node
func ListAgentStates(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allAgents []agents.Agent

	allPagesAgents, err := agents.List(exporter.Client, agents.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allAgents, err = agents.ExtractAgents(allPagesAgents)
	if err != nil {
		return err
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

	return nil
}

// ListNetworks : Count total number of instantiated Networks
func ListNetworks(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allNetworks []networks.Network

	allPagesNetworks, err := networks.List(exporter.Client, networks.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allNetworks, err = networks.ExtractNetworks(allPagesNetworks)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["networks"].Metric,
		prometheus.GaugeValue, float64(len(allNetworks)))

	return nil
}

// ListSecGroups : count total number of instantiated Security Groups
func ListSecGroups(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allSecurityGroups []groups.SecGroup

	allPagesSecurityGroups, err := groups.List(exporter.Client, groups.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allSecurityGroups, err = groups.ExtractGroups(allPagesSecurityGroups)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["security_groups"].Metric,
		prometheus.GaugeValue, float64(len(allSecurityGroups)))

	return nil
}

// ListSubnets : count total number of instantiated Subnets
func ListSubnets(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allSubnets []subnets.Subnet

	allPagesSubnets, err := subnets.List(exporter.Client, subnets.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allSubnets, err = subnets.ExtractSubnets(allPagesSubnets)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["subnets"].Metric,
		prometheus.GaugeValue, float64(len(allSubnets)))

	return nil
}

// PortBinding represents a port which includes port bindings
type PortBinding struct {
	ports.Port
	portsbinding.PortsBindingExt
}

// ListPorts generates metrics about ports inside the OpenStack cloud
func ListPorts(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allPorts []PortBinding

	allPagesPorts, err := ports.List(exporter.Client, ports.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	err = ports.ExtractPortsInto(allPagesPorts, &allPorts)
	if err != nil {
		return err
	}

	portsWithNoIP := float64(0)
	lbaasPortsInactive := float64(0)

	for _, port := range allPorts {
		if port.Status == "ACTIVE" && len(port.FixedIPs) == 0 {
			portsWithNoIP++
		}

		if port.DeviceOwner == "neutron:LOADBALANCERV2" && port.Status != "ACTIVE" {
			lbaasPortsInactive++
		}

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["port"].Metric,
			prometheus.GaugeValue, 1, port.ID, port.NetworkID, port.MACAddress, port.DeviceOwner, port.Status, port.VIFType)
	}

	// NOTE(mnaser): We should deprecate this and users can replace it by
	//               count(openstack_neutron_port)
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["ports"].Metric,
		prometheus.GaugeValue, float64(len(allPorts)))

	// NOTE(mnaser): We should deprecate this and users can replace it by:
	//               count(openstack_neutron_port{device_owner="neutron:LOADBALANCERV2",status!="ACTIVE"})
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["ports_lb_not_active"].Metric,
		prometheus.GaugeValue, lbaasPortsInactive)

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["ports_no_ips"].Metric,
		prometheus.GaugeValue, portsWithNoIP)

	return nil
}

// ListNetworkIPAvailabilities : count total number of used IPs per Network
func ListNetworkIPAvailabilities(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allNetworkIPAvailabilities []networkipavailabilities.NetworkIPAvailability

	allPagesNetworkIPAvailabilities, err := networkipavailabilities.List(exporter.Client, networkipavailabilities.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allNetworkIPAvailabilities, err = networkipavailabilities.ExtractNetworkIPAvailabilities(allPagesNetworkIPAvailabilities)
	if err != nil {
		return err
	}

	for _, NetworkIPAvailabilities := range allNetworkIPAvailabilities {
		for _, SubnetIPAvailability := range NetworkIPAvailabilities.SubnetIPAvailabilities {
			totalIPs, err := strconv.ParseFloat(SubnetIPAvailability.TotalIPs, 64)
			if err != nil {
				return err
			}
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["network_ip_availabilities_total"].Metric,
				prometheus.GaugeValue, totalIPs, NetworkIPAvailabilities.NetworkID,
				NetworkIPAvailabilities.NetworkName, strconv.Itoa(SubnetIPAvailability.IPVersion), SubnetIPAvailability.CIDR,
				SubnetIPAvailability.SubnetName, NetworkIPAvailabilities.ProjectID)

			usedIPs, err := strconv.ParseFloat(SubnetIPAvailability.UsedIPs, 64)
			if err != nil {
				return err
			}
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["network_ip_availabilities_used"].Metric,
				prometheus.GaugeValue, usedIPs, NetworkIPAvailabilities.NetworkID,
				NetworkIPAvailabilities.NetworkName, strconv.Itoa(SubnetIPAvailability.IPVersion), SubnetIPAvailability.CIDR,
				SubnetIPAvailability.SubnetName, NetworkIPAvailabilities.ProjectID)
		}
	}

	return nil
}

// ListRouters : count total number of instantiated Routers and those that are not in ACTIVE state
func ListRouters(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allRouters []routers.Router

	allPagesRouters, err := routers.List(exporter.Client, routers.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allRouters, err = routers.ExtractRouters(allPagesRouters)
	if err != nil {
		return err
	}

	failedRouters := 0
	for _, router := range allRouters {
		if router.Status != "ACTIVE" {
			failedRouters = failedRouters + 1
		}
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["routers"].Metric,
		prometheus.GaugeValue, float64(len(allRouters)))
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["routers_not_active"].Metric,
		prometheus.GaugeValue, float64(failedRouters))

	return nil
}

// ListLBs : count total number of instantiated LoadBalancers and those that are not in ACTIVE state
func ListLBs(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allLBs []loadbalancers.LoadBalancer

	allPagesLBs, err := loadbalancers.List(exporter.Client, loadbalancers.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allLBs, err = loadbalancers.ExtractLoadBalancers(allPagesLBs)
	if err != nil {
		return err
	}

	failedLBs := 0
	for _, lb := range allLBs {
		if lb.ProvisioningStatus != "ACTIVE" {
			failedLBs = failedLBs + 1
		}
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["loadbalancers"].Metric,
		prometheus.GaugeValue, float64(len(allLBs)))
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["loadbalancers_not_active"].Metric,
		prometheus.GaugeValue, float64(failedLBs))

	return nil
}
