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

	gophercloud "github.com/gophercloud/gophercloud/v2"
	neutronexts "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/agents"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/mtu"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/networkipavailabilities"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/provider"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/quotas"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/endpointgroups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/ikepolicies"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/ipsecpolicies"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/services"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/siteconnections"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterTypedExporter("network", NewNeutronExporter)
}

var knownNetworkStatuses = map[string]int{
	"ACTIVE": 0,
	"BUILD":  1,
	"DOWN":   2,
	"ERROR":  3,
}

func mapNetworkStatus(current string) int {
	return mapStatus(knownNetworkStatuses, current)
}

var knownVpnStatuses = map[string]int{
	"ACTIVE":         0,
	"DOWN":           1,
	"BUILD":          2,
	"ERROR":          3,
	"PENDING_CREATE": 4,
	"PENDING_UPDATE": 5,
	"PENDING_DELETE": 6,
}

func mapVpnServiceStatus(current string) int {
	return mapStatus(knownVpnStatuses, current)
}

func mapVpnConnectionStatus(current string) int {
	return mapStatus(knownVpnStatuses, current)
}

// NeutronExporter : extends BaseOpenStackExporter
type NeutronExporter struct {
	BaseOpenStackExporter
	sched Schedule
	descs neutronDescs
}

type neutronDescs struct {
	FloatingIPs                    *prometheus.Desc `metric:"floating_ips"`
	FloatingIPsAssociatedNotActive *prometheus.Desc `metric:"floating_ips_associated_not_active"`
	FloatingIP                     *prometheus.Desc `metric:"floating_ip"                     labels:"id,floating_network_id,router_id,status,project_id,floating_ip_address"`
	Networks                       *prometheus.Desc `metric:"networks"`
	Network                        *prometheus.Desc `metric:"network"                         labels:"id,tenant_id,status,name,is_shared,is_external,provider_network_type,provider_physical_network,provider_segmentation_id,subnets,tags,mtu"`
	SecurityGroups                 *prometheus.Desc `metric:"security_groups"`
	Subnets                        *prometheus.Desc `metric:"subnets"`
	Subnet                         *prometheus.Desc `metric:"subnet"                          labels:"id,tenant_id,name,network_id,cidr,gateway_ip,enable_dhcp,dns_nameservers,tags"`
	Port                           *prometheus.Desc `metric:"port"                            labels:"uuid,network_id,mac_address,device_owner,device_id,status,binding_vif_type,admin_state_up,fixed_ips"`
	Ports                          *prometheus.Desc `metric:"ports"`
	PortsNoIPs                     *prometheus.Desc `metric:"ports_no_ips"`
	PortsLBNotActive               *prometheus.Desc `metric:"ports_lb_not_active"`
	Router                         *prometheus.Desc `metric:"router"                          labels:"id,name,project_id,admin_state_up,status,external_network_id"`
	Routers                        *prometheus.Desc `metric:"routers"`
	RoutersNotActive               *prometheus.Desc `metric:"routers_not_active"`
	L3AgentOfRouter                *prometheus.Desc `metric:"l3_agent_of_router"              labels:"router_id,l3_agent_id,ha_state,agent_alive,agent_admin_up,agent_host"`
	AgentState                     *prometheus.Desc `metric:"agent_state"                     labels:"id,hostname,service,adminState,availability_zone"`
	VpnEndpointGroups              *prometheus.Desc `metric:"vpn_endpoint_groups"`
	VpnIKEPolicies                 *prometheus.Desc `metric:"vpn_ike_policies"`
	VpnIPsecPolicies               *prometheus.Desc `metric:"vpn_ipsec_policies"`
	VpnServices                    *prometheus.Desc `metric:"vpn_services"`
	VpnService                     *prometheus.Desc `metric:"vpn_service"                     labels:"id,project_id,subnet_id,router_id,admin_state_up,name,external_ipv4,external_ipv6,flavor_id"`
	VpnSiteConnections             *prometheus.Desc `metric:"vpn_siteconnections"`
	VpnSiteConnection              *prometheus.Desc `metric:"vpn_siteconnection"              labels:"id,project_id,admin_state_up,name,vpn_service_id,ike_policy_id,ipsec_policy_id,peer_id,peer_ep_group_id,local_id,local_ep_group_id"`
	NetworkIPAvailabilitiesTotal   *prometheus.Desc `metric:"network_ip_availabilities_total" labels:"network_id,network_name,ip_version,cidr,subnet_name,project_id"`
	NetworkIPAvailabilitiesUsed    *prometheus.Desc `metric:"network_ip_availabilities_used"  labels:"network_id,network_name,ip_version,cidr,subnet_name,project_id"`
	SubnetsTotal                   *prometheus.Desc `metric:"subnets_total"                   labels:"ip_version,prefix,prefix_length,project_id,subnet_pool_id,subnet_pool_name"`
	SubnetsUsed                    *prometheus.Desc `metric:"subnets_used"                    labels:"ip_version,prefix,prefix_length,project_id,subnet_pool_id,subnet_pool_name"`
	SubnetsFree                    *prometheus.Desc `metric:"subnets_free"                    labels:"ip_version,prefix,prefix_length,project_id,subnet_pool_id,subnet_pool_name"`
	QuotaNetwork                   *prometheus.Desc `metric:"quota_network"                   labels:"type,tenant,tenant_id"  slow:"true"`
	QuotaSubnet                    *prometheus.Desc `metric:"quota_subnet"                    labels:"type,tenant,tenant_id"  slow:"true"`
	QuotaSubnetPool                *prometheus.Desc `metric:"quota_subnetpool"                labels:"type,tenant,tenant_id"  slow:"true"`
	QuotaPort                      *prometheus.Desc `metric:"quota_port"                      labels:"type,tenant,tenant_id"  slow:"true"`
	QuotaRouter                    *prometheus.Desc `metric:"quota_router"                    labels:"type,tenant,tenant_id"  slow:"true"`
	QuotaFloatingIP                *prometheus.Desc `metric:"quota_floatingip"                labels:"type,tenant,tenant_id"  slow:"true"`
	QuotaSecurityGroup             *prometheus.Desc `metric:"quota_security_group"            labels:"type,tenant,tenant_id"  slow:"true"`
	QuotaSecurityGroupRule         *prometheus.Desc `metric:"quota_security_group_rule"       labels:"type,tenant,tenant_id"  slow:"true"`
	QuotaRBACPolicy                *prometheus.Desc `metric:"quota_rbac_policy"               labels:"type,tenant,tenant_id"  slow:"true"`
}

type neutronNetworkExt struct {
	networks.Network
	external.NetworkExternalExt
	provider.NetworkProviderExt
	mtu.NetworkMTUExt
}

type neutronPortExt struct {
	ports.Port
	portsbinding.PortsBindingExt
}

type neutronScrape struct {
	floatingIPs         []floatingips.FloatingIP
	allNetworks         []neutronNetworkExt
	securityGroups      []groups.SecGroup
	allSubnets          []subnets.Subnet
	allPorts            []neutronPortExt
	allRouters          []routers.Router
	ovnBackend          bool
	vpnEndpointGroups   []endpointgroups.EndpointGroup
	vpnIKEPolicies      []ikepolicies.Policy
	vpnIPsecPolicies    []ipsecpolicies.Policy
	vpnServices         []services.Service
	vpnSiteConnections  []siteconnections.Connection
	allAgents           []agents.Agent
	netIPAvailabilities []neutronNetIPAvail
	subnetPools         []subnetpoolWithSubnets
}

type neutronNetIPAvail struct {
	NetworkID   string
	NetworkName string
	ProjectID   string
	Subnets     []neutronSubnetIPAvail
}

type neutronSubnetIPAvail struct {
	SubnetName string
	CIDR       string
	IPVersion  int
	TotalIPs   float64
	UsedIPs    float64
}

var neutronGraph = Graph[*NeutronExporter, neutronScrape]{
	Sources: []Source[*NeutronExporter, neutronScrape]{
		{Name: "floatingips", Fetch: (*NeutronExporter).fetchFloatingIPs},
		{Name: "networks", Fetch: (*NeutronExporter).fetchNetworks},
		{Name: "secgroups", Fetch: (*NeutronExporter).fetchSecGroups},
		{Name: "subnets", Fetch: (*NeutronExporter).fetchSubnets},
		{Name: "ports", Fetch: (*NeutronExporter).fetchPorts},
		{Name: "routers", Fetch: (*NeutronExporter).fetchRouters},
		{Name: "vpn_endpoint_groups", Fetch: (*NeutronExporter).fetchVpnEndpointGroups},
		{Name: "vpn_ike_policies", Fetch: (*NeutronExporter).fetchVpnIKEPolicies},
		{Name: "vpn_ipsec_policies", Fetch: (*NeutronExporter).fetchVpnIPsecPolicies},
		{Name: "vpn_services", Fetch: (*NeutronExporter).fetchVpnServices},
		{Name: "vpn_siteconnections", Fetch: (*NeutronExporter).fetchVpnSiteConnections},
		{Name: "agents", Fetch: (*NeutronExporter).fetchAgents},
		{Name: "net_ip_avail", Fetch: (*NeutronExporter).fetchNetIPAvail},
		{Name: "subnet_pools", DependsOn: []string{"subnets"}, Fetch: (*NeutronExporter).fetchSubnetPools},
	},
	Emitters: []Emitter[*NeutronExporter, neutronScrape]{
		{Name: "floatingips", Metrics: []string{"floating_ips", "floating_ips_associated_not_active", "floating_ip"}, Sources: []string{"floatingips"}, Emit: (*NeutronExporter).emitFloatingIPs},
		{Name: "networks", Metrics: []string{"networks", "network"}, Sources: []string{"networks"}, Emit: (*NeutronExporter).emitNetworks},
		{Name: "secgroups", Metrics: []string{"security_groups"}, Sources: []string{"secgroups"}, Emit: (*NeutronExporter).emitSecGroups},
		{Name: "subnets", Metrics: []string{"subnets", "subnet"}, Sources: []string{"subnets"}, Emit: (*NeutronExporter).emitSubnets},
		{Name: "ports", Metrics: []string{"port", "ports", "ports_no_ips", "ports_lb_not_active"}, Sources: []string{"ports"}, Emit: (*NeutronExporter).emitPorts},
		{Name: "routers", Metrics: []string{"router", "routers", "routers_not_active", "l3_agent_of_router"}, Sources: []string{"routers"}, Emit: (*NeutronExporter).emitRouters},
		{Name: "vpn_endpoint_groups", Metrics: []string{"vpn_endpoint_groups"}, Sources: []string{"vpn_endpoint_groups"}, Emit: (*NeutronExporter).emitVpnEndpointGroups},
		{Name: "vpn_ike_policies", Metrics: []string{"vpn_ike_policies"}, Sources: []string{"vpn_ike_policies"}, Emit: (*NeutronExporter).emitVpnIKEPolicies},
		{Name: "vpn_ipsec_policies", Metrics: []string{"vpn_ipsec_policies"}, Sources: []string{"vpn_ipsec_policies"}, Emit: (*NeutronExporter).emitVpnIPsecPolicies},
		{Name: "vpn_services", Metrics: []string{"vpn_services", "vpn_service"}, Sources: []string{"vpn_services"}, Emit: (*NeutronExporter).emitVpnServices},
		{Name: "vpn_siteconnections", Metrics: []string{"vpn_siteconnections", "vpn_siteconnection"}, Sources: []string{"vpn_siteconnections"}, Emit: (*NeutronExporter).emitVpnSiteConnections},
		{Name: "agents", Metrics: []string{"agent_state"}, Sources: []string{"agents"}, Emit: (*NeutronExporter).emitAgents},
		{Name: "net_ip_avail", Metrics: []string{"network_ip_availabilities_total", "network_ip_availabilities_used"}, Sources: []string{"net_ip_avail"}, Emit: (*NeutronExporter).emitNetIPAvail},
		{Name: "subnet_pools", Metrics: []string{"subnets_total", "subnets_used", "subnets_free"}, Sources: []string{"subnet_pools", "subnets"}, Emit: (*NeutronExporter).emitSubnetPools},
		{Name: "quotas", Metrics: []string{"quota_network", "quota_subnet", "quota_subnetpool", "quota_port", "quota_router", "quota_floatingip", "quota_security_group", "quota_security_group_rule", "quota_rbac_policy"}, Sources: []string{}, Emit: (*NeutronExporter).emitQuotas},
	},
}

// vpnaasMetrics lists all metric names owned by the VPNaaS sub-service.
var vpnaasMetrics = []string{
	"vpn_endpoint_groups",
	"vpn_ike_policies",
	"vpn_ipsec_policies",
	"vpn_services",
	"vpn_service",
	"vpn_siteconnections",
	"vpn_siteconnection",
}

func NewNeutronExporter(config *ExporterConfig, logger *slog.Logger) (*NeutronExporter, error) {
	e := &NeutronExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "neutron",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	// Probe for VPNaaS extension; disable its metrics if absent so PruneSchedule
	// drops the entire VPN subgraph without ever making VPN API calls.
	_, err := neutronexts.Get(context.Background(), config.ClientV2, "vpnaas").Extract()
	if err != nil {
		if gophercloud.ResponseCodeIs(err, 404) {
			logger.Info("VPNaaS extension not available, disabling VPN metrics", "exporter", "neutron")
			for _, m := range vpnaasMetrics {
				e.DisabledMetrics = append(e.DisabledMetrics, e.qualifiedMetricName(m))
			}
		} else {
			logger.Warn("VPNaaS extension probe failed, assuming available", "exporter", "neutron", "err", err)
		}
	}
	e.RegisterAndFillDescs(&e.descs)
	sched, err := neutronGraph.PruneSchedule(&e.BaseOpenStackExporter)
	if err != nil {
		return nil, err
	}
	e.sched = sched
	neutronGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)
	return e, nil
}

func (e *NeutronExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(neutronScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &neutronGraph, e.sched, s, ch)
	})
}

// --- Sources ---

func (e *NeutronExporter) fetchFloatingIPs(ctx context.Context, s *neutronScrape) error {
	allPages, err := floatingips.List(e.ClientV2, floatingips.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.floatingIPs, err = floatingips.ExtractFloatingIPs(allPages)
	return err
}

func (e *NeutronExporter) fetchNetworks(ctx context.Context, s *neutronScrape) error {
	allPages, err := networks.List(e.ClientV2, networks.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	return networks.ExtractNetworksInto(allPages, &s.allNetworks)
}

func (e *NeutronExporter) fetchSecGroups(ctx context.Context, s *neutronScrape) error {
	allPages, err := groups.List(e.ClientV2, groups.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.securityGroups, err = groups.ExtractGroups(allPages)
	return err
}

func (e *NeutronExporter) fetchSubnets(ctx context.Context, s *neutronScrape) error {
	allPages, err := subnets.List(e.ClientV2, subnets.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.allSubnets, err = subnets.ExtractSubnets(allPages)
	return err
}

func (e *NeutronExporter) fetchPorts(ctx context.Context, s *neutronScrape) error {
	allPages, err := ports.List(e.ClientV2, ports.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	return ports.ExtractPortsInto(allPages, &s.allPorts)
}

func (e *NeutronExporter) fetchRouters(ctx context.Context, s *neutronScrape) error {
	allPages, err := routers.List(e.ClientV2, routers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	var err2 error
	s.allRouters, err2 = routers.ExtractRouters(allPages)
	if err2 != nil {
		return err2
	}
	ovnPages, err2 := agents.List(e.ClientV2, agents.ListOpts{Binary: "ovn-controller"}).AllPages(ctx)
	if err2 != nil {
		return err2
	}
	ovnAgents, err2 := agents.ExtractAgents(ovnPages)
	if err2 != nil {
		return err2
	}
	s.ovnBackend = len(ovnAgents) > 0
	return nil
}

func (e *NeutronExporter) fetchVpnEndpointGroups(ctx context.Context, s *neutronScrape) error {
	allPages, err := endpointgroups.List(e.ClientV2, endpointgroups.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.vpnEndpointGroups, err = endpointgroups.ExtractEndpointGroups(allPages)
	return err
}

func (e *NeutronExporter) fetchVpnIKEPolicies(ctx context.Context, s *neutronScrape) error {
	allPages, err := ikepolicies.List(e.ClientV2, ikepolicies.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.vpnIKEPolicies, err = ikepolicies.ExtractPolicies(allPages)
	return err
}

func (e *NeutronExporter) fetchVpnIPsecPolicies(ctx context.Context, s *neutronScrape) error {
	allPages, err := ipsecpolicies.List(e.ClientV2, ipsecpolicies.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.vpnIPsecPolicies, err = ipsecpolicies.ExtractPolicies(allPages)
	return err
}

func (e *NeutronExporter) fetchVpnServices(ctx context.Context, s *neutronScrape) error {
	allPages, err := services.List(e.ClientV2, services.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.vpnServices, err = services.ExtractServices(allPages)
	return err
}

func (e *NeutronExporter) fetchVpnSiteConnections(ctx context.Context, s *neutronScrape) error {
	allPages, err := siteconnections.List(e.ClientV2, siteconnections.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.vpnSiteConnections, err = siteconnections.ExtractConnections(allPages)
	return err
}

func (e *NeutronExporter) fetchAgents(ctx context.Context, s *neutronScrape) error {
	allPages, err := agents.List(e.ClientV2, agents.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.allAgents, err = agents.ExtractAgents(allPages)
	return err
}

func (e *NeutronExporter) fetchNetIPAvail(ctx context.Context, s *neutronScrape) error {
	type customSubnetIPAvailability struct {
		SubnetName string      `json:"subnet_name"`
		CIDR       string      `json:"cidr"`
		IPVersion  int         `json:"ip_version"`
		TotalIPs   json.Number `json:"total_ips"`
		UsedIPs    json.Number `json:"used_ips"`
	}
	type customNetworkIPAvailability struct {
		NetworkID              string                       `json:"network_id"`
		NetworkName            string                       `json:"network_name"`
		ProjectID              string                       `json:"project_id"`
		TenantID               string                       `json:"tenant_id"`
		SubnetIPAvailabilities []customSubnetIPAvailability `json:"subnet_ip_availability"`
	}
	type availabilityWrapper struct {
		NetworkIPAvailabilities []customNetworkIPAvailability `json:"network_ip_availabilities"`
	}

	allPages, err := networkipavailabilities.List(e.ClientV2, networkipavailabilities.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	body := allPages.GetBody()
	bodyMap, ok := body.(map[string]interface{})
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
	for _, net := range wrapper.NetworkIPAvailabilities {
		projectID := net.ProjectID
		if projectID == "" {
			projectID = net.TenantID
		}
		entry := neutronNetIPAvail{NetworkID: net.NetworkID, NetworkName: net.NetworkName, ProjectID: projectID}
		for _, subnet := range net.SubnetIPAvailabilities {
			totalBig := new(big.Float)
			if _, ok := totalBig.SetString(subnet.TotalIPs.String()); !ok {
				return fmt.Errorf("failed to parse total IPs: %s", subnet.TotalIPs.String())
			}
			totalFloat64, _ := totalBig.Float64()
			usedBig := new(big.Float)
			if _, ok = usedBig.SetString(subnet.UsedIPs.String()); !ok {
				return fmt.Errorf("failed to parse used IPs: %s", subnet.UsedIPs.String())
			}
			usedFloat64, _ := usedBig.Float64()
			entry.Subnets = append(entry.Subnets, neutronSubnetIPAvail{
				SubnetName: subnet.SubnetName,
				CIDR:       subnet.CIDR,
				IPVersion:  subnet.IPVersion,
				TotalIPs:   totalFloat64,
				UsedIPs:    usedFloat64,
			})
		}
		s.netIPAvailabilities = append(s.netIPAvailabilities, entry)
	}
	return nil
}

func (e *NeutronExporter) fetchSubnetPools(ctx context.Context, s *neutronScrape) error {
	allPagesSubnetPools, err := subnetpools.List(e.ClientV2, subnetpools.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	allSubnetPools, err := subnetpools.ExtractSubnetPools(allPagesSubnetPools)
	if err != nil {
		return err
	}
	s.subnetPools, err = subnetpoolsWithSubnets(allSubnetPools, s.allSubnets)
	return err
}

// --- Emitters ---

func (e *NeutronExporter) emitFloatingIPs(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	failedFIPs := 0
	for _, fip := range s.floatingIPs {
		emitGauge(ch, e.descs.FloatingIP, 1, fip.ID, fip.FloatingNetworkID, fip.RouterID, fip.Status, fip.ProjectID, fip.FloatingIP)
		if fip.FixedIP != "" && fip.Status != "ACTIVE" {
			failedFIPs++
		}
	}
	emitGauge(ch, e.descs.FloatingIPs, float64(len(s.floatingIPs)))
	emitGauge(ch, e.descs.FloatingIPsAssociatedNotActive, float64(failedFIPs))
	return nil
}

func (e *NeutronExporter) emitNetworks(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.Networks, float64(len(s.allNetworks)))
	for _, net := range s.allNetworks {
		emitGauge(ch, e.descs.Network,
			float64(mapNetworkStatus(net.Status)), net.ID, net.TenantID, net.Status, net.Name,
			strconv.FormatBool(net.Shared), strconv.FormatBool(net.External), net.NetworkType,
			net.PhysicalNetwork, net.SegmentationID, strings.Join(net.Subnets, ","), strings.Join(net.Tags, ","), strconv.Itoa(net.MTU))
	}
	return nil
}

func (e *NeutronExporter) emitSecGroups(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.SecurityGroups, float64(len(s.securityGroups)))
	return nil
}

func (e *NeutronExporter) emitSubnets(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.Subnets, float64(len(s.allSubnets)))
	for _, subnet := range s.allSubnets {
		emitGauge(ch, e.descs.Subnet,
			1.0, subnet.ID, subnet.TenantID, subnet.Name, subnet.NetworkID, subnet.CIDR,
			subnet.GatewayIP, strconv.FormatBool(subnet.EnableDHCP), strings.Join(subnet.DNSNameservers, ","), strings.Join(subnet.Tags, ","))
	}
	return nil
}

func (e *NeutronExporter) emitPorts(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	portsNoIP := float64(0)
	lbaasPortsInactive := float64(0)
	for _, port := range s.allPorts {
		if port.Status == "ACTIVE" && len(port.FixedIPs) == 0 {
			portsNoIP++
		}
		if port.DeviceOwner == "neutron:LOADBALANCERV2" && port.Status != "ACTIVE" {
			lbaasPortsInactive++
		}
		if e.descs.Port != nil {
			fixedIPs := ""
			n := len(port.FixedIPs)
			if n == 1 {
				fixedIPs = port.FixedIPs[0].IPAddress
			} else if n > 1 {
				for idx, fip := range port.FixedIPs {
					if idx != 0 {
						fixedIPs += ","
					}
					fixedIPs += fip.IPAddress
				}
			}
			emitGauge(ch, e.descs.Port,
				1, port.ID, port.NetworkID, port.MACAddress, port.DeviceOwner, port.DeviceID,
				port.Status, port.VIFType, strconv.FormatBool(port.AdminStateUp), fixedIPs)
		}
	}
	emitGauge(ch, e.descs.Ports, float64(len(s.allPorts)))
	emitGauge(ch, e.descs.PortsLBNotActive, lbaasPortsInactive)
	emitGauge(ch, e.descs.PortsNoIPs, portsNoIP)
	return nil
}

func (e *NeutronExporter) emitRouters(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	failedRouters := 0
	for _, router := range s.allRouters {
		if router.Status != "ACTIVE" {
			failedRouters++
		}
		emitGauge(ch, e.descs.Router, 1, router.ID, router.Name, router.ProjectID,
			strconv.FormatBool(router.AdminStateUp), router.Status, router.GatewayInfo.NetworkID)
		if s.ovnBackend {
			continue
		}
		if e.descs.L3AgentOfRouter != nil {
			allPagesL3Agents, err := routers.ListL3Agents(e.ClientV2, router.ID).AllPages(ctx)
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
				emitGauge(ch, e.descs.L3AgentOfRouter,
					float64(state), router.ID, agent.ID,
					agent.HAState, strconv.FormatBool(agent.Alive), strconv.FormatBool(agent.AdminStateUp), agent.Host)
			}
		}
	}
	emitGauge(ch, e.descs.Routers, float64(len(s.allRouters)))
	emitGauge(ch, e.descs.RoutersNotActive, float64(failedRouters))
	return nil
}

func (e *NeutronExporter) emitVpnEndpointGroups(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.VpnEndpointGroups, float64(len(s.vpnEndpointGroups)))
	return nil
}

func (e *NeutronExporter) emitVpnIKEPolicies(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.VpnIKEPolicies, float64(len(s.vpnIKEPolicies)))
	return nil
}

func (e *NeutronExporter) emitVpnIPsecPolicies(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.VpnIPsecPolicies, float64(len(s.vpnIPsecPolicies)))
	return nil
}

func (e *NeutronExporter) emitVpnServices(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.VpnServices, float64(len(s.vpnServices)))
	for _, svc := range s.vpnServices {
		emitGauge(ch, e.descs.VpnService,
			float64(mapVpnServiceStatus(svc.Status)),
			svc.ID, svc.ProjectID, svc.SubnetID, svc.RouterID, strconv.FormatBool(svc.AdminStateUp),
			svc.Name, svc.ExternalV4IP, svc.ExternalV6IP, svc.FlavorID)
	}
	return nil
}

func (e *NeutronExporter) emitVpnSiteConnections(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.VpnSiteConnections, float64(len(s.vpnSiteConnections)))
	for _, conn := range s.vpnSiteConnections {
		emitGauge(ch, e.descs.VpnSiteConnection,
			float64(mapVpnConnectionStatus(conn.Status)),
			conn.ID, conn.ProjectID, strconv.FormatBool(conn.AdminStateUp), conn.Name,
			conn.VPNServiceID, conn.IKEPolicyID, conn.IPSecPolicyID, conn.PeerID,
			conn.PeerEPGroupID, conn.LocalID, conn.LocalEPGroupID)
	}
	return nil
}

func (e *NeutronExporter) emitAgents(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	for _, agent := range s.allAgents {
		state := 0
		if agent.Alive {
			state = 1
		}
		adminState := "down"
		if agent.AdminStateUp {
			adminState = "up"
		}
		id := agent.ID
		if id == "" {
			var err error
			if id, err = e.UUIDGenFunc(); err != nil {
				return err
			}
		}
		emitGauge(ch, e.descs.AgentState,
			float64(state), id, agent.Host, agent.Binary, adminState, agent.AvailabilityZone)
	}
	return nil
}

func (e *NeutronExporter) emitNetIPAvail(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	for _, net := range s.netIPAvailabilities {
		for _, subnet := range net.Subnets {
			emitGauge(ch, e.descs.NetworkIPAvailabilitiesTotal, subnet.TotalIPs, net.NetworkID, net.NetworkName, strconv.Itoa(subnet.IPVersion), subnet.CIDR, subnet.SubnetName, net.ProjectID)
			emitGauge(ch, e.descs.NetworkIPAvailabilitiesUsed, subnet.UsedIPs, net.NetworkID, net.NetworkName, strconv.Itoa(subnet.IPVersion), subnet.CIDR, subnet.SubnetName, net.ProjectID)
		}
	}
	return nil
}

func (e *NeutronExporter) emitSubnetPools(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	for _, pool := range s.subnetPools {
		ipPrefixes, err := pool.IPPrefixes()
		if err != nil {
			return err
		}
		for _, ipPrefix := range ipPrefixes {
			for prefixLength := pool.MinPrefixLen; prefixLength <= pool.MaxPrefixLen; prefixLength++ {
				if prefixLength < int(ipPrefix.Bits()) {
					continue
				}
				totalSubnets := math.Pow(2, float64(prefixLength-int(ipPrefix.Bits())))
				emitGauge(ch, e.descs.SubnetsTotal, totalSubnets, strconv.Itoa(pool.IPversion), ipPrefix.String(), strconv.Itoa(prefixLength),
					pool.ProjectID, pool.ID, pool.Name)
				usedSubnets := calculateUsedSubnets(pool.subnets, ipPrefix, prefixLength)
				emitGauge(ch, e.descs.SubnetsUsed, usedSubnets, strconv.Itoa(pool.IPversion), ipPrefix.String(), strconv.Itoa(prefixLength),
					pool.ProjectID, pool.ID, pool.Name)
				if e.descs.SubnetsFree != nil {
					freeSubnets, err := calculateFreeSubnets(&ipPrefix, pool.subnets, prefixLength)
					if err != nil {
						return err
					}
					emitGauge(ch, e.descs.SubnetsFree, freeSubnets, strconv.Itoa(pool.IPversion), ipPrefix.String(), strconv.Itoa(prefixLength),
						pool.ProjectID, pool.ID, pool.Name)
				}
			}
		}
	}
	return nil
}

func (e *NeutronExporter) emitQuotas(ctx context.Context, s *neutronScrape, ch chan<- prometheus.Metric) error {
	allProjects, err := GetProjects(ctx, &e.BaseOpenStackExporter)
	if err != nil {
		return err
	}
	for _, p := range allProjects {
		quota, err := quotas.GetDetail(ctx, e.ClientV2, p.ID).Extract()
		if err != nil {
			return err
		}
		e.emitNeutronQuotaDetail(ch, e.descs.QuotaNetwork, quota.Network, p.Name, p.ID)
		e.emitNeutronQuotaDetail(ch, e.descs.QuotaSubnet, quota.Subnet, p.Name, p.ID)
		e.emitNeutronQuotaDetail(ch, e.descs.QuotaSubnetPool, quota.SubnetPool, p.Name, p.ID)
		e.emitNeutronQuotaDetail(ch, e.descs.QuotaPort, quota.Port, p.Name, p.ID)
		e.emitNeutronQuotaDetail(ch, e.descs.QuotaRouter, quota.Router, p.Name, p.ID)
		e.emitNeutronQuotaDetail(ch, e.descs.QuotaFloatingIP, quota.FloatingIP, p.Name, p.ID)
		e.emitNeutronQuotaDetail(ch, e.descs.QuotaSecurityGroup, quota.SecurityGroup, p.Name, p.ID)
		e.emitNeutronQuotaDetail(ch, e.descs.QuotaSecurityGroupRule, quota.SecurityGroupRule, p.Name, p.ID)
		e.emitNeutronQuotaDetail(ch, e.descs.QuotaRBACPolicy, quota.RBACPolicy, p.Name, p.ID)
	}
	return nil
}

func (e *NeutronExporter) emitNeutronQuotaDetail(ch chan<- prometheus.Metric, desc *prometheus.Desc, q quotas.QuotaDetail, projectName, projectID string) {
	emitGauge(ch, desc, float64(q.Used), "used", projectName, projectID)
	emitGauge(ch, desc, float64(q.Reserved), "reserved", projectName, projectID)
	emitGauge(ch, desc, float64(q.Limit), "limit", projectName, projectID)
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
