package exporters

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"net/netip"
	"strconv"
	"strings"

	"go4.org/netipx"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/agents"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/networkipavailabilities"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/provider"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/quotas"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/prometheus/client_golang/prometheus"
)

var knownNetworkStatuses = map[string]int{
	"ACTIVE": 0,
	"BUILD":  1,
	"DOWN":   2,
	"ERROR":  3,
}

func mapNetworkStatus(current string) int {
	return mapStatus(knownNetworkStatuses, current)
}

// NeutronExporter : extends BaseOpenStackExporter
type NeutronExporter struct {
	BaseOpenStackExporter
}

var (
	defaultNeutronNetIPsLabels  = []string{"network_id", "network_name", "ip_version", "cidr", "subnet_name", "project_id"}
	defaultNeutronSubnetsLabels = []string{"ip_version", "prefix", "prefix_length", "project_id", "subnet_pool_id", "subnet_pool_name"}
	defaultNeutronQuotaLabels   = []string{"type", "tenant", "tenant_id"}
)

var defaultNeutronMetrics = []Metric{
	{Name: "floating_ips", Fn: ListFloatingIps},
	{Name: "floating_ips_associated_not_active"},
	{Name: "floating_ip", Labels: []string{"id", "floating_network_id", "router_id", "status", "project_id", "floating_ip_address"}},
	{Name: "networks", Fn: ListNetworks},
	{Name: "network", Labels: []string{"id", "tenant_id", "status", "name", "is_shared", "is_external", "provider_network_type",
		"provider_physical_network", "provider_segmentation_id", "subnets", "tags"}},
	{Name: "security_groups", Fn: ListSecGroups},
	{Name: "subnets", Fn: ListSubnets},
	{Name: "subnet", Labels: []string{"id", "tenant_id", "name", "network_id", "cidr", "gateway_ip", "enable_dhcp", "dns_nameservers", "tags"}},
	{Name: "port", Labels: []string{"uuid", "network_id", "mac_address", "device_owner", "device_id", "status", "binding_vif_type", "admin_state_up", "fixed_ips"}, Fn: ListPorts},
	{Name: "ports"},
	{Name: "ports_no_ips"},
	{Name: "ports_lb_not_active"},
	{Name: "router", Labels: []string{"id", "name", "project_id", "admin_state_up", "status", "external_network_id"}},
	{Name: "routers", Fn: ListRouters},
	{Name: "routers_not_active"},
	{Name: "l3_agent_of_router", Labels: []string{"router_id", "l3_agent_id", "ha_state", "agent_alive", "agent_admin_up", "agent_host"}},
	{Name: "agent_state", Labels: []string{"id", "hostname", "service", "adminState", "availability_zone"}, Fn: ListAgentStates},
	{Name: "network_ip_availabilities_total", Labels: defaultNeutronNetIPsLabels, Fn: ListNetworkIPAvailabilities},
	{Name: "network_ip_availabilities_used", Labels: defaultNeutronNetIPsLabels},
	{Name: "subnets_total", Labels: defaultNeutronSubnetsLabels, Fn: ListSubnetsPerPool},
	{Name: "subnets_used", Labels: defaultNeutronSubnetsLabels},
	{Name: "subnets_free", Labels: defaultNeutronSubnetsLabels},
	{Name: "quota_network", Labels: defaultNeutronQuotaLabels, Fn: ListNetworkQuotas, Slow: true},
	{Name: "quota_subnet", Labels: defaultNeutronQuotaLabels, Fn: nil, Slow: true},
	{Name: "quota_subnetpool", Labels: defaultNeutronQuotaLabels, Fn: nil, Slow: true},
	{Name: "quota_port", Labels: defaultNeutronQuotaLabels, Fn: nil, Slow: true},
	{Name: "quota_router", Labels: defaultNeutronQuotaLabels, Fn: nil, Slow: true},
	{Name: "quota_floatingip", Labels: defaultNeutronQuotaLabels, Fn: nil, Slow: true},
	{Name: "quota_security_group", Labels: defaultNeutronQuotaLabels, Fn: nil, Slow: true},
	{Name: "quota_security_group_rule", Labels: defaultNeutronQuotaLabels, Fn: nil, Slow: true},
	{Name: "quota_rbac_policy", Labels: defaultNeutronQuotaLabels, Fn: nil, Slow: true},
}

// NewNeutronExporter : returns a pointer to NeutronExporter
func NewNeutronExporter(config *ExporterConfig, logger *slog.Logger) (*NeutronExporter, error) {
	exporter := NeutronExporter{
		BaseOpenStackExporter{
			Name:           "neutron",
			ExporterConfig: *config,
			logger:         logger,
		},
	}

	for _, metric := range defaultNeutronMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}

	return &exporter, nil
}

// ListFloatingIps : count total number of instantiated FloatingIPs and those that are associated to private IP but not in ACTIVE state
func ListFloatingIps(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allFloatingIPs []floatingips.FloatingIP

	allPagesFloatingIPs, err := floatingips.List(exporter.ClientV2, floatingips.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	allFloatingIPs, err = floatingips.ExtractFloatingIPs(allPagesFloatingIPs)
	if err != nil {
		return err
	}

	failedFIPs := 0
	for _, fip := range allFloatingIPs {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["floating_ip"].Metric,
			prometheus.GaugeValue, 1, fip.ID, fip.FloatingNetworkID, fip.RouterID, fip.Status, fip.ProjectID, fip.FloatingIP)

		if fip.FixedIP != "" && fip.Status != "ACTIVE" {
			failedFIPs = failedFIPs + 1
		}
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["floating_ips"].Metric,
		prometheus.GaugeValue, float64(len(allFloatingIPs)))
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["floating_ips_associated_not_active"].Metric,
		prometheus.GaugeValue, float64(failedFIPs))

	return nil
}

// ListAgentStates : list agent state per node
func ListAgentStates(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allAgents []agents.Agent

	allPagesAgents, err := agents.List(exporter.ClientV2, agents.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	allAgents, err = agents.ExtractAgents(allPagesAgents)
	if err != nil {
		return err
	}

	for _, agent := range allAgents {
		var state = 0
		var id string
		var zone string

		if agent.Alive {
			state = 1
		}

		adminState := "down"
		if agent.AdminStateUp {
			adminState = "up"
		}

		id = agent.ID
		if id == "" {
			if id, err = exporter.UUIDGenFunc(); err != nil {
				return err
			}
		}

		zone = agent.AvailabilityZone

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["agent_state"].Metric,
			prometheus.CounterValue, float64(state), id, agent.Host, agent.Binary, adminState, zone)
	}

	return nil
}

// ListNetworks : Count total number of instantiated Networks and list each Network info
func ListNetworks(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	type NetworkWithExt struct {
		networks.Network
		external.NetworkExternalExt
		provider.NetworkProviderExt
	}

	var allNetworks []NetworkWithExt

	allPagesNetworks, err := networks.List(exporter.ClientV2, networks.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	err = networks.ExtractNetworksInto(allPagesNetworks, &allNetworks)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["networks"].Metric,
		prometheus.GaugeValue, float64(len(allNetworks)))

	if !exporter.MetricIsDisabled("network") {
		for _, net := range allNetworks {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["network"].Metric,
				prometheus.GaugeValue, float64(mapNetworkStatus(net.Status)), net.ID, net.TenantID, net.Status, net.Name,
				strconv.FormatBool(net.Shared), strconv.FormatBool(net.External), net.NetworkType,
				net.PhysicalNetwork, net.SegmentationID, strings.Join(net.Subnets, ","), strings.Join(net.Tags, ","))
		}
	}

	return nil
}

// ListSecGroups : count total number of instantiated Security Groups
func ListSecGroups(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allSecurityGroups []groups.SecGroup

	allPagesSecurityGroups, err := groups.List(exporter.ClientV2, groups.ListOpts{}).AllPages(ctx)
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

// ListSubnets : count total number of instantiated Subnets and list each Subnet info
func ListSubnets(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allSubnets []subnets.Subnet

	allPagesSubnets, err := subnets.List(exporter.ClientV2, subnets.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	allSubnets, err = subnets.ExtractSubnets(allPagesSubnets)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["subnets"].Metric,
		prometheus.GaugeValue, float64(len(allSubnets)))

	if !exporter.MetricIsDisabled("subnet") {
		for _, subnet := range allSubnets {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["subnet"].Metric,
				prometheus.GaugeValue, 1.0, subnet.ID, subnet.TenantID, subnet.Name, subnet.NetworkID, subnet.CIDR,
				subnet.GatewayIP, strconv.FormatBool(subnet.EnableDHCP), strings.Join(subnet.DNSNameservers, ","), strings.Join(subnet.Tags, ","))
		}
	}

	return nil
}

// ListPorts generates metrics about ports inside the OpenStack cloud
func ListPorts(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	type PortBinding struct {
		ports.Port
		portsbinding.PortsBindingExt
	}

	var allPorts []PortBinding

	allPagesPorts, err := ports.List(exporter.ClientV2, ports.ListOpts{}).AllPages(ctx)
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

		if !exporter.MetricIsDisabled("port") {
			var fixedIPs = ""

			portFixedIPsLen := len(port.FixedIPs)
			if portFixedIPsLen == 1 {
				fixedIPs = port.FixedIPs[0].IPAddress
			} else if portFixedIPsLen > 1 {
				for idx, fip := range port.FixedIPs {
					// Joining IPs into a string with ',' separator
					if idx != 0 {
						fixedIPs += ","
					}
					fixedIPs += fip.IPAddress
				}
			}

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["port"].Metric,
				prometheus.GaugeValue, 1, port.ID, port.NetworkID, port.MACAddress, port.DeviceOwner, port.DeviceID,
				port.Status, port.VIFType, strconv.FormatBool(port.AdminStateUp), fixedIPs)
		}
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
func ListNetworkIPAvailabilities(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {

	type CustomSubnetIPAvailability struct {
		SubnetName string      `json:"subnet_name"`
		CIDR       string      `json:"cidr"`
		IPVersion  int         `json:"ip_version"`
		TotalIPs   json.Number `json:"total_ips"`
		UsedIPs    json.Number `json:"used_ips"`
	}

	type CustomNetworkIPAvailability struct {
		NetworkID              string                       `json:"network_id"`
		NetworkName            string                       `json:"network_name"`
		ProjectID              string                       `json:"project_id"`
		TenantID               string                       `json:"tenant_id"`
		SubnetIPAvailabilities []CustomSubnetIPAvailability `json:"subnet_ip_availability"`
	}

	type availabilityWrapper struct {
		NetworkIPAvailabilities []CustomNetworkIPAvailability `json:"network_ip_availabilities"`
	}

	// Getting raw response
	allPages, err := networkipavailabilities.List(exporter.ClientV2, networkipavailabilities.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	// Decode raw JSON manually to avoid gophercloud unmarshaling big.Int error
	body := allPages.GetBody()
	bodyMap, ok := body.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected type for body: %T", body)
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return fmt.Errorf("failed to marshal body back to JSON: %w", err)
	}

	var wrapper availabilityWrapper
	if err := json.Unmarshal(bodyBytes, &wrapper); err != nil {
		return fmt.Errorf("failed to unmarshal network_ip_availabilities JSON: %w", err)
	}

	for _, network := range wrapper.NetworkIPAvailabilities {
		projectID := network.ProjectID
		if projectID == "" && network.TenantID != "" {
			projectID = network.TenantID
		}

		for _, subnet := range network.SubnetIPAvailabilities {
			// Use big.Float to parse TotalIPs
			totalBig := new(big.Float)
			_, ok := totalBig.SetString(subnet.TotalIPs.String())
			if !ok {
				return fmt.Errorf("failed to parse total IPs: %s", subnet.TotalIPs.String())
			}
			totalFloat64, _ := totalBig.Float64()

			usedBig := new(big.Float)
			_, ok = usedBig.SetString(subnet.UsedIPs.String())
			if !ok {
				return fmt.Errorf("failed to parse used IPs: %s", subnet.UsedIPs.String())
			}
			usedFloat64, _ := usedBig.Float64()

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["network_ip_availabilities_total"].Metric,
				prometheus.GaugeValue, totalFloat64, network.NetworkID,
				network.NetworkName, strconv.Itoa(subnet.IPVersion), subnet.CIDR,
				subnet.SubnetName, projectID)

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["network_ip_availabilities_used"].Metric,
				prometheus.GaugeValue, usedFloat64, network.NetworkID,
				network.NetworkName, strconv.Itoa(subnet.IPVersion), subnet.CIDR,
				subnet.SubnetName, projectID)
		}
	}

	return nil
}

// ListRouters : count total number of instantiated Routers and those that are not in ACTIVE state
func ListRouters(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allRouters []routers.Router
	// We need to know if neutron has ovn backend
	var ovnBackendEnabled = false

	allPagesRouters, err := routers.List(exporter.ClientV2, routers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	allRouters, err = routers.ExtractRouters(allPagesRouters)
	if err != nil {
		return err
	}

	// Requesting Neutron network-agents with binary='ovn-controller'
	ovnAgentsPages, err := agents.List(exporter.ClientV2, agents.ListOpts{Binary: "ovn-controller"}).AllPages(ctx)
	if err != nil {
		return err
	}

	ovnAgents, err := agents.ExtractAgents(ovnAgentsPages)
	if err != nil {
		return err
	}

	// If we have received data, then OVN is neutron network backend.
	if len(ovnAgents) > 0 {
		ovnBackendEnabled = true
	}

	failedRouters := 0
	for _, router := range allRouters {
		if router.Status != "ACTIVE" {
			failedRouters++
		}

		if !exporter.MetricIsDisabled("router") {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["router"].Metric,
				prometheus.GaugeValue, 1, router.ID, router.Name, router.ProjectID,
				strconv.FormatBool(router.AdminStateUp), router.Status, router.GatewayInfo.NetworkID)
		}

		if ovnBackendEnabled {
			continue
			// Because ovn-backend doesn't have router l3-agent entity
		}

		if !exporter.MetricIsDisabled("l3_agent_of_router") {
			allPagesL3Agents, err := routers.ListL3Agents(exporter.ClientV2, router.ID).AllPages(ctx)
			if err != nil {
				return err
			}

			l3Agents, err := routers.ExtractL3Agents(allPagesL3Agents)
			if err != nil {
				return err
			}

			for _, agent := range l3Agents {
				state := 0
				if agent.Alive {
					state = 1
				}

				ch <- prometheus.MustNewConstMetric(exporter.Metrics["l3_agent_of_router"].Metric,
					prometheus.GaugeValue, float64(state), router.ID, agent.ID,
					agent.HAState, strconv.FormatBool(agent.Alive), strconv.FormatBool(agent.AdminStateUp), agent.Host)
			}
		}
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["routers"].Metric,
		prometheus.GaugeValue, float64(len(allRouters)))
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["routers_not_active"].Metric,
		prometheus.GaugeValue, float64(failedRouters))

	return nil
}

// subnetpoolWithSubnets : subnetpools.SubnetPool augmented with its subnets
type subnetpoolWithSubnets struct {
	subnetpools.SubnetPool
	subnets []netip.Prefix
}

// IPPrefixes : returns a subnetpoolWithSubnets's prefixes converted to netip.Prefix structs.
func (s *subnetpoolWithSubnets) IPPrefixes() ([]netip.Prefix, error) {
	result := make([]netip.Prefix, len(s.Prefixes))

	for i, prefix := range s.Prefixes {
		ipPrefix, err := netip.ParsePrefix(prefix)
		if err != nil {
			return nil, err
		}

		result[i] = ipPrefix
	}

	return result, nil
}

// subnetpoolsWithSubnets : builds a slice of subnetpoolWithSubnets from subnetpools.SubnetPool and subnets.Subnet structs
func subnetpoolsWithSubnets(pools []subnetpools.SubnetPool, subnets []subnets.Subnet) ([]subnetpoolWithSubnets, error) {
	subnetPrefixes := make(map[string][]netip.Prefix)

	for _, subnet := range subnets {
		if subnet.SubnetPoolID != "" {
			subnetPrefix, err := netip.ParsePrefix(subnet.CIDR)
			if err != nil {
				return nil, err
			}

			subnetPrefixes[subnet.SubnetPoolID] = append(subnetPrefixes[subnet.SubnetPoolID], subnetPrefix)
		}
	}

	result := make([]subnetpoolWithSubnets, len(pools))
	for i, pool := range pools {
		result[i] = subnetpoolWithSubnets{pool, subnetPrefixes[pool.ID]}
	}

	return result, nil
}

// calculateFreeSubnets : Count how many CIDRs of length prefixLength there are in poolPrefix after removing subnetsInPool
func calculateFreeSubnets(poolPrefix *netip.Prefix, subnetsInPool []netip.Prefix, prefixLength int) (float64, error) {
	builder := netipx.IPSetBuilder{}
	builder.AddPrefix(*poolPrefix)

	for _, subnet := range subnetsInPool {
		builder.RemovePrefix(subnet)
	}

	ipset, err := builder.IPSet()
	if err != nil {
		return 0, err
	}

	count := 0.0
	for _, prefix := range ipset.Prefixes() {
		if int(prefix.Bits()) > prefixLength {
			continue
		}

		count += math.Pow(2, float64(prefixLength-int(prefix.Bits())))
	}

	return count, nil
}

// calculateUsedSubnets : find all subnets that overlap with ipPrefix and count the different subnet sizes.
// Finally, return the count that matches prefixLength.
func calculateUsedSubnets(subnets []netip.Prefix, ipPrefix netip.Prefix, prefixLength int) float64 {
	result := make(map[int]int)

	for _, subnet := range subnets {
		if !ipPrefix.Overlaps(subnet) {
			continue
		}

		result[int(subnet.Bits())]++
	}

	return float64(result[prefixLength])
}

// ListSubnetsPerPool : Count used/free/total number of subnets per subnet pool
func ListSubnetsPerPool(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	allPagesSubnets, err := subnets.List(exporter.ClientV2, subnets.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	allSubnets, err := subnets.ExtractSubnets(allPagesSubnets)
	if err != nil {
		return err
	}

	allPagesSubnetPools, err := subnetpools.List(exporter.ClientV2, subnetpools.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	allSubnetPools, err := subnetpools.ExtractSubnetPools(allPagesSubnetPools)
	if err != nil {
		return err
	}

	subnetPools, err := subnetpoolsWithSubnets(allSubnetPools, allSubnets)
	if err != nil {
		return err
	}

	for _, subnetPool := range subnetPools {
		ipPrefixes, err := subnetPool.IPPrefixes()
		if err != nil {
			return err
		}

		for _, ipPrefix := range ipPrefixes {
			for prefixLength := subnetPool.MinPrefixLen; prefixLength <= subnetPool.MaxPrefixLen; prefixLength++ {
				if prefixLength < int(ipPrefix.Bits()) {
					continue
				}

				totalSubnets := math.Pow(2, float64(prefixLength-int(ipPrefix.Bits())))
				ch <- prometheus.MustNewConstMetric(exporter.Metrics["subnets_total"].Metric,
					prometheus.GaugeValue, totalSubnets, strconv.Itoa(subnetPool.IPversion), ipPrefix.String(), strconv.Itoa(prefixLength),
					subnetPool.ProjectID, subnetPool.ID, subnetPool.Name)

				usedSubnets := calculateUsedSubnets(subnetPool.subnets, ipPrefix, prefixLength)
				ch <- prometheus.MustNewConstMetric(exporter.Metrics["subnets_used"].Metric,
					prometheus.GaugeValue, usedSubnets, strconv.Itoa(subnetPool.IPversion), ipPrefix.String(), strconv.Itoa(prefixLength),
					subnetPool.ProjectID, subnetPool.ID, subnetPool.Name)

				freeSubnets, err := calculateFreeSubnets(&ipPrefix, subnetPool.subnets, prefixLength)
				if err != nil {
					return err
				}

				ch <- prometheus.MustNewConstMetric(exporter.Metrics["subnets_free"].Metric,
					prometheus.GaugeValue, freeSubnets, strconv.Itoa(subnetPool.IPversion), ipPrefix.String(), strconv.Itoa(prefixLength),
					subnetPool.ProjectID, subnetPool.ID, subnetPool.Name)
			}
		}
	}

	return nil
}

func collectNeutronQuotaDetail(ch chan<- prometheus.Metric, metric *prometheus.Desc, q quotas.QuotaDetail, projectName, projectID string) {
	ch <- prometheus.MustNewConstMetric(metric, prometheus.GaugeValue, float64(q.Used), "used", projectName, projectID)
	ch <- prometheus.MustNewConstMetric(metric, prometheus.GaugeValue, float64(q.Reserved), "reserved", projectName, projectID)
	ch <- prometheus.MustNewConstMetric(metric, prometheus.GaugeValue, float64(q.Limit), "limit", projectName, projectID)
}

func ListNetworkQuotas(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allProjects []projects.Project

	cli, err := newIdentityV3ClientV2FromExporter(exporter, "network")
	if err != nil {
		return err
	}

	allPagesProject, err := projects.List(cli, projects.ListOpts{DomainID: exporter.DomainID}).AllPages(ctx)
	if err != nil {
		return err
	}

	allProjects, err = projects.ExtractProjects(allPagesProject)
	if err != nil {
		return err
	}

	for _, p := range allProjects {
		// quota are obtained from the neutron API, so now we can just use this exporter's client
		quota, err := quotas.GetDetail(ctx, exporter.ClientV2, p.ID).Extract()
		if err != nil {
			return err
		}

		collectNeutronQuotaDetail(ch, exporter.Metrics["quota_network"].Metric, quota.Network, p.Name, p.ID)
		collectNeutronQuotaDetail(ch, exporter.Metrics["quota_subnet"].Metric, quota.Subnet, p.Name, p.ID)
		collectNeutronQuotaDetail(ch, exporter.Metrics["quota_subnetpool"].Metric, quota.SubnetPool, p.Name, p.ID)
		collectNeutronQuotaDetail(ch, exporter.Metrics["quota_port"].Metric, quota.Port, p.Name, p.ID)
		collectNeutronQuotaDetail(ch, exporter.Metrics["quota_router"].Metric, quota.Router, p.Name, p.ID)
		collectNeutronQuotaDetail(ch, exporter.Metrics["quota_floatingip"].Metric, quota.FloatingIP, p.Name, p.ID)
		collectNeutronQuotaDetail(ch, exporter.Metrics["quota_security_group"].Metric, quota.SecurityGroup, p.Name, p.ID)
		collectNeutronQuotaDetail(ch, exporter.Metrics["quota_security_group_rule"].Metric, quota.SecurityGroupRule, p.Name, p.ID)
		collectNeutronQuotaDetail(ch, exporter.Metrics["quota_rbac_policy"].Metric, quota.RBACPolicy, p.Name, p.ID)

	}

	return nil
}
