package exporters

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/aggregates"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/availabilityzones"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/hypervisors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/limits"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/quotasets"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/secgroups"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/services"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/usage"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// Latest supported microversion for Nova which provides all metrics
// See also: https://github.com/openstack-exporter/openstack-exporter/issues/249
const novaLatestSupportedMicroversion = "2.87"

var knownServerStatuses = map[string]int{
	"ACTIVE":            0,
	"BUILD":             1,  // The server has not finished the original build process.
	"BUILD(spawning)":   2,  // The server has not finished the original build process but networking works (HP Cloud specific)
	"DELETED":           3,  // The server is deleted.
	"ERROR":             4,  // The server is in error.
	"HARD_REBOOT":       5,  // The server is hard rebooting.
	"PASSWORD":          6,  // The password is being reset on the server.
	"REBOOT":            7,  // The server is in a soft reboot state.
	"REBUILD":           8,  // The server is currently being rebuilt from an image.
	"RESCUE":            9,  // The server is in rescue mode.
	"RESIZE":            10, // Server is performing the differential copy of data that changed during its initial copy.
	"SHUTOFF":           11, // The virtual machine (VM) was powered down by the user, but not through the OpenStack Compute API.
	"SUSPENDED":         12, // The server is suspended, either by request or necessity.
	"UNKNOWN":           13, // The state of the server is unknown. Contact your cloud provider.
	"VERIFY_RESIZE":     14, // System is awaiting confirmation that the server is operational after a move or resize.
	"MIGRATING":         15, // The server is migrating. This is caused by a live migration (moving a server that is active) action.
	"PAUSED":            16, // The server is paused.
	"REVERT_RESIZE":     17, // The resize or migration of a server failed for some reason. The destination server is being cleaned up and the original source server is restarting.
	"SHELVED":           18, // The server is in shelved state. Depends on the shelve offload time, the server will be automatically shelved off loaded.
	"SHELVED_OFFLOADED": 19, // The shelved server is offloaded (removed from the compute host) and it needs unshelved action to be used again.
	"SOFT_DELETED":      20, // The server is marked as deleted but will remain in the cloud for some configurable amount of time.
}

func mapServerStatus(current string) int {
	return mapStatus(knownServerStatuses, current)
}

type NovaExporter struct {
	BaseOpenStackExporter
}

var (
	defaultNovaServerStatusLabels = []string{"id", "status", "name", "tenant_id", "user_id", "address_ipv4",
		"address_ipv6", "host_id", "hypervisor_hostname", "uuid", "availability_zone", "flavor_id", "instance_libvirt"}

	defaultNovaHypervisorLabels = []string{"hostname", "availability_zone", "aggregates"}
	defaultNovaLimitsLabels     = []string{"tenant", "tenant_id"}
	defaultNovaQuotaLabels      = []string{"type", "tenant", "tenant_id"}
)

var defaultNovaMetrics = []Metric{
	{Name: "flavors", Fn: ListFlavors},
	{Name: "flavor", Labels: []string{"id", "name", "vcpus", "ram", "disk", "is_public"}},
	{Name: "availability_zones", Fn: ListAZs},
	{Name: "security_groups", Fn: ListComputeSecGroups},
	{Name: "total_vms", Fn: ListAllServers},
	{Name: "agent_state", Labels: []string{"id", "hostname", "service", "adminState", "zone", "disabledReason"}, Fn: ListNovaAgentState},
	{Name: "running_vms", Labels: defaultNovaHypervisorLabels, Fn: ListHypervisors},
	{Name: "current_workload", Labels: defaultNovaHypervisorLabels},
	{Name: "vcpus_available", Labels: defaultNovaHypervisorLabels},
	{Name: "vcpus_used", Labels: defaultNovaHypervisorLabels},
	{Name: "memory_available_bytes", Labels: defaultNovaHypervisorLabels},
	{Name: "memory_used_bytes", Labels: defaultNovaHypervisorLabels},
	{Name: "local_storage_available_bytes", Labels: defaultNovaHypervisorLabels},
	{Name: "local_storage_used_bytes", Labels: defaultNovaHypervisorLabels},
	{Name: "free_disk_bytes", Labels: defaultNovaHypervisorLabels},
	{Name: "server_status", Labels: defaultNovaServerStatusLabels},
	{Name: "limits_vcpus_max", Labels: defaultNovaLimitsLabels, Fn: ListComputeLimits, Slow: true},
	{Name: "limits_vcpus_used", Labels: defaultNovaLimitsLabels},
	{Name: "limits_memory_max", Labels: defaultNovaLimitsLabels},
	{Name: "limits_memory_used", Labels: defaultNovaLimitsLabels},
	{Name: "limits_instances_used", Labels: defaultNovaLimitsLabels},
	{Name: "limits_instances_max", Labels: defaultNovaLimitsLabels},
	{Name: "server_local_gb", Labels: []string{"name", "id", "tenant_id"}, Fn: ListUsage, Slow: true},
	{Name: "quota_cores", Labels: defaultNovaQuotaLabels, Fn: ListQuotas},
	{Name: "quota_instances", Labels: defaultNovaQuotaLabels},
	{Name: "quota_key_pairs", Labels: defaultNovaQuotaLabels},
	{Name: "quota_metadata_items", Labels: defaultNovaQuotaLabels},
	{Name: "quota_ram", Labels: defaultNovaQuotaLabels},
	{Name: "quota_server_groups", Labels: defaultNovaQuotaLabels},
	{Name: "quota_server_group_members", Labels: defaultNovaQuotaLabels},
	{Name: "quota_fixed_ips", Labels: defaultNovaQuotaLabels},
	{Name: "quota_floating_ips", Labels: defaultNovaQuotaLabels},
	{Name: "quota_security_group_rules", Labels: defaultNovaQuotaLabels},
	{Name: "quota_security_groups", Labels: defaultNovaQuotaLabels},
	{Name: "quota_injected_file_content_bytes", Labels: defaultNovaQuotaLabels},
	{Name: "quota_injected_file_path_bytes", Labels: defaultNovaQuotaLabels},
	{Name: "quota_injected_files", Labels: defaultNovaQuotaLabels},
}

func NewNovaExporter(config *ExporterConfig, logger *slog.Logger) (*NovaExporter, error) {
	ctx := context.TODO()

	err := utils.SetupClientMicroversionV2(ctx, config.ClientV2, "OS_COMPUTE_API_VERSION", novaLatestSupportedMicroversion, logger)
	if err != nil {
		return nil, err
	}

	exporter := NovaExporter{
		BaseOpenStackExporter{
			Name:           "nova",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	for _, metric := range defaultNovaMetrics {
		if metric.Name == "server_status" {
			metric.Labels = append(defaultNovaServerStatusLabels, config.NovaMetadataMapping.Labels...)
		}
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}

	return &exporter, nil
}

func ListNovaAgentState(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allServices []services.Service

	allPagesServices, err := services.List(exporter.ClientV2, services.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	if allServices, err = services.ExtractServices(allPagesServices); err != nil {
		return err
	}

	for _, service := range allServices {
		var state = 0
		if service.State == "up" {
			state = 1
		}
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["agent_state"].Metric,
			prometheus.CounterValue, float64(state), service.ID, service.Host, service.Binary, service.Status, service.Zone, service.DisabledReason)
	}

	return nil
}

func ListHypervisors(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allHypervisors []hypervisors.Hypervisor
	var allAggregates []aggregates.Aggregate
	var listOpts *hypervisors.ListOpts

	// See also: https://github.com/openstack-exporter/openstack-exporter/pull/520
	apiMv, _ := strconv.ParseFloat(exporter.ClientV2.Microversion, 64)
	if apiMv >= 2.33 {
		limit := 1000 // NOTE: default paging limit is 1k
		listOpts = &hypervisors.ListOpts{Limit: &limit}
	} else {
		listOpts = &hypervisors.ListOpts{}
	}

	allPagesHypervisors, err := hypervisors.List(exporter.ClientV2, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}

	allHypervisors, err = hypervisors.ExtractHypervisors(allPagesHypervisors)
	if err != nil {
		return err
	}

	allPagesAggregates, err := aggregates.List(exporter.ClientV2).AllPages(ctx)
	if err != nil {
		return err
	}

	allAggregates, err = aggregates.ExtractAggregates(allPagesAggregates)
	if err != nil {
		return err
	}

	hostToAzMap := map[string]string{}     // map of hypervisors and in which AZ they are
	hostToAggrMap := map[string][]string{} // map of hypervisors and of which aggregates they are part of
	for _, a := range allAggregates {
		isAzAggregate := isAzAggregate(a)
		for _, h := range a.Hosts {
			// Map the AZ of this aggregate to each host part of this aggregate
			if a.AvailabilityZone != "" {
				hostToAzMap[h] = a.AvailabilityZone
			}
			// Map the aggregate name to each host part of this aggregate
			if !isAzAggregate {
				hostToAggrMap[h] = append(hostToAggrMap[h], a.Name)
			}
		}
	}

	for _, hypervisor := range allHypervisors {
		availabilityZone := ""
		if val, ok := hostToAzMap[hypervisor.Service.Host]; ok {
			availabilityZone = val
		}
		aggregates := aggregatesLabel(hypervisor.Service.Host, hostToAggrMap)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["running_vms"].Metric,
			prometheus.GaugeValue, float64(hypervisor.RunningVMs), hypervisor.HypervisorHostname, availabilityZone, aggregates)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["current_workload"].Metric,
			prometheus.GaugeValue, float64(hypervisor.CurrentWorkload), hypervisor.HypervisorHostname, availabilityZone, aggregates)

		var vcpus int
		if !reflect.ValueOf(hypervisor.CPUInfo).IsZero() {
			vcpus = max(hypervisor.CPUInfo.Topology.Cells, 1) * hypervisor.CPUInfo.Topology.Sockets * hypervisor.CPUInfo.Topology.Cores * hypervisor.CPUInfo.Topology.Threads
		} else {
			vcpus = hypervisor.VCPUs
		}
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["vcpus_available"].Metric,
			prometheus.GaugeValue, float64(vcpus), hypervisor.HypervisorHostname, availabilityZone, aggregates)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["vcpus_used"].Metric,
			prometheus.GaugeValue, float64(hypervisor.VCPUsUsed), hypervisor.HypervisorHostname, availabilityZone, aggregates)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["memory_available_bytes"].Metric,
			prometheus.GaugeValue, float64(hypervisor.MemoryMB*MEGABYTE), hypervisor.HypervisorHostname, availabilityZone, aggregates)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["memory_used_bytes"].Metric,
			prometheus.GaugeValue, float64(hypervisor.MemoryMBUsed*MEGABYTE), hypervisor.HypervisorHostname, availabilityZone, aggregates)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["local_storage_available_bytes"].Metric,
			prometheus.GaugeValue, float64(hypervisor.LocalGB*GIGABYTE), hypervisor.HypervisorHostname, availabilityZone, aggregates)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["local_storage_used_bytes"].Metric,
			prometheus.GaugeValue, float64(hypervisor.LocalGBUsed*GIGABYTE), hypervisor.HypervisorHostname, availabilityZone, aggregates)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["free_disk_bytes"].Metric,
			prometheus.GaugeValue, float64(hypervisor.FreeDiskGB*GIGABYTE), hypervisor.HypervisorHostname, availabilityZone, aggregates)

	}

	return nil
}

func ListFlavors(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allFlavors []flavors.Flavor

	allPagesFlavors, err := flavors.ListDetail(exporter.ClientV2, flavors.ListOpts{AccessType: "None"}).AllPages(ctx)
	if err != nil {
		return err
	}

	allFlavors, err = flavors.ExtractFlavors(allPagesFlavors)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["flavors"].Metric,
		prometheus.GaugeValue, float64(len(allFlavors)))
	for _, f := range allFlavors {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["flavor"].Metric,
			prometheus.GaugeValue, 1, f.ID, f.Name, fmt.Sprintf("%v", f.VCPUs), fmt.Sprintf("%v", f.RAM), fmt.Sprintf("%v", f.Disk), fmt.Sprintf("%v", f.IsPublic))
	}

	return nil
}

func collectNovaQuotaDetail(ch chan<- prometheus.Metric, metric *prometheus.Desc, q quotasets.QuotaDetail, projectName, projectID string) {
	ch <- prometheus.MustNewConstMetric(metric, prometheus.GaugeValue, float64(q.InUse), "in_use", projectName, projectID)
	ch <- prometheus.MustNewConstMetric(metric, prometheus.GaugeValue, float64(q.Reserved), "reserved", projectName, projectID)
	ch <- prometheus.MustNewConstMetric(metric, prometheus.GaugeValue, float64(q.Limit), "limit", projectName, projectID)
}

func ListQuotas(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allProjects []projects.Project

	cli, err := newIdentityV3ClientV2FromExporter(exporter, "compute")
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
		quotaSet, err := quotasets.GetDetail(ctx, exporter.ClientV2, p.ID).Extract()
		if err != nil {
			return err
		}

		collectNovaQuotaDetail(ch, exporter.Metrics["quota_cores"].Metric, quotaSet.Cores, p.Name, p.ID)
		collectNovaQuotaDetail(ch, exporter.Metrics["quota_instances"].Metric, quotaSet.Instances, p.Name, p.ID)
		collectNovaQuotaDetail(ch, exporter.Metrics["quota_key_pairs"].Metric, quotaSet.KeyPairs, p.Name, p.ID)
		collectNovaQuotaDetail(ch, exporter.Metrics["quota_metadata_items"].Metric, quotaSet.MetadataItems, p.Name, p.ID)
		collectNovaQuotaDetail(ch, exporter.Metrics["quota_ram"].Metric, quotaSet.RAM, p.Name, p.ID)
		collectNovaQuotaDetail(ch, exporter.Metrics["quota_server_groups"].Metric, quotaSet.ServerGroups, p.Name, p.ID)
		collectNovaQuotaDetail(ch, exporter.Metrics["quota_server_group_members"].Metric, quotaSet.ServerGroupMembers, p.Name, p.ID)
		collectNovaQuotaDetail(ch, exporter.Metrics["quota_fixed_ips"].Metric, quotaSet.FixedIPs, p.Name, p.ID)
		collectNovaQuotaDetail(ch, exporter.Metrics["quota_floating_ips"].Metric, quotaSet.FloatingIPs, p.Name, p.ID)
		collectNovaQuotaDetail(ch, exporter.Metrics["quota_security_group_rules"].Metric, quotaSet.SecurityGroupRules, p.Name, p.ID)
		collectNovaQuotaDetail(ch, exporter.Metrics["quota_security_groups"].Metric, quotaSet.SecurityGroups, p.Name, p.ID)
		collectNovaQuotaDetail(ch, exporter.Metrics["quota_injected_file_content_bytes"].Metric, quotaSet.InjectedFileContentBytes, p.Name, p.ID)
		collectNovaQuotaDetail(ch, exporter.Metrics["quota_injected_file_path_bytes"].Metric, quotaSet.InjectedFilePathBytes, p.Name, p.ID)
		collectNovaQuotaDetail(ch, exporter.Metrics["quota_injected_files"].Metric, quotaSet.InjectedFiles, p.Name, p.ID)

	}
	return nil
}

func ListAZs(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allAZs []availabilityzones.AvailabilityZone

	allPagesAZs, err := availabilityzones.List(exporter.ClientV2).AllPages(ctx)
	if err != nil {
		return err
	}

	if allAZs, err = availabilityzones.ExtractAvailabilityZones(allPagesAZs); err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["availability_zones"].Metric,
		prometheus.GaugeValue, float64(len(allAZs)))

	return nil
}

func ListComputeSecGroups(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allSecurityGroups []secgroups.SecurityGroup

	allPagesSecurityGroups, err := secgroups.List(exporter.ClientV2).AllPages(ctx)
	if err != nil {
		return err
	}

	if allSecurityGroups, err = secgroups.ExtractSecurityGroups(allPagesSecurityGroups); err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["security_groups"].Metric,
		prometheus.GaugeValue, float64(len(allSecurityGroups)))

	return nil
}

func ListAllServers(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	type ServerWithExt = servers.Server

	var allServers []ServerWithExt
	var serverListOption servers.ListOpts
	var flavorIDMapper flavorIDMapper

	if exporter.TenantID == "" {
		serverListOption = servers.ListOpts{AllTenants: true}
	} else {
		serverListOption = servers.ListOpts{TenantID: exporter.TenantID}

	}
	allPagesServers, err := servers.List(exporter.ClientV2, serverListOption).AllPages(ctx)
	if err != nil {
		return err
	}

	err = servers.ExtractServersInto(allPagesServers, &allServers)
	if err != nil {
		return err
	}

	// check if flavor.id present, if not - we probably got response like for 2.46+
	var mapperRequired bool
	for _, server := range allServers {
		if _, ok := server.Flavor["id"]; !ok {
			mapperRequired = true
			break
		}
	}

	apiMv, _ := strconv.ParseFloat(exporter.ClientV2.Microversion, 64)
	if apiMv >= 2.46 || mapperRequired {
		// https://docs.openstack.org/api-ref/compute/#list-servers-detailed
		// ***
		// If micro-version is greater than 2.46,
		// we need to retrieve all flavors once again and search for flavor_id by name,
		// as flavor_id are only available in server's detail data up to that version.
		flavorIDMapper, err = newFlavorIDMapper(ctx, exporter.ClientV2)
		if err != nil {
			return err
		}
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["total_vms"].Metric,
		prometheus.GaugeValue, float64(len(allServers)))

	// Server status metrics
	if !exporter.MetricIsDisabled("server_status") {
		for _, server := range allServers {
			var flavorID string
			if flavorIDMapper == nil {
				flavorID = fmt.Sprintf("%v", server.Flavor["id"])
			} else {
				flavorID = flavorIDMapper.Search(server.Flavor["original_name"])
			}

			labelValues := []string{
				server.ID, server.Status, server.Name, server.TenantID,
				server.UserID, server.AccessIPv4, server.AccessIPv6, server.HostID, server.HypervisorHostname, server.ID,
				server.AvailabilityZone, flavorID, server.InstanceName,
			}
			metadataValues := exporter.NovaMetadataMapping.Extract(server.Metadata)

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["server_status"].Metric,
				prometheus.GaugeValue, float64(mapServerStatus(server.Status)), append(labelValues, metadataValues...)...)
		}
	}
	return nil
}

func ListComputeLimits(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allProjects []projects.Project

	cli, err := newIdentityV3ClientV2FromExporter(exporter, "compute")
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
		// Limits are obtained from the nova API, so now we can just use this exporter's client
		limits, err := limits.Get(ctx, exporter.ClientV2, limits.GetOpts{TenantID: p.ID}).Extract()
		if err != nil {
			return err
		}

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["limits_vcpus_max"].Metric,
			prometheus.GaugeValue, float64(limits.Absolute.MaxTotalCores), p.Name, p.ID)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["limits_vcpus_used"].Metric,
			prometheus.GaugeValue, float64(limits.Absolute.TotalCoresUsed), p.Name, p.ID)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["limits_memory_max"].Metric,
			prometheus.GaugeValue, float64(limits.Absolute.MaxTotalRAMSize), p.Name, p.ID)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["limits_memory_used"].Metric,
			prometheus.GaugeValue, float64(limits.Absolute.TotalRAMUsed), p.Name, p.ID)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["limits_instances_used"].Metric,
			prometheus.GaugeValue, float64(limits.Absolute.TotalInstancesUsed), p.Name, p.ID)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["limits_instances_max"].Metric,
			prometheus.GaugeValue, float64(limits.Absolute.MaxTotalInstances), p.Name, p.ID)
	}

	return nil
}

// ListUsage add metrics about usage
func ListUsage(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	allPagesUsage, err := usage.AllTenants(exporter.ClientV2, usage.AllTenantsOpts{Detailed: true}).AllPages(ctx)
	if err != nil {
		return err
	}

	allTenantsUsage, err := usage.ExtractAllTenants(allPagesUsage)
	if err != nil {
		return err
	}

	// Server status metrics
	for _, tenant := range allTenantsUsage {
		for _, server := range tenant.ServerUsages {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["server_local_gb"].Metric,
				prometheus.GaugeValue, float64(server.LocalGB), server.Name, server.InstanceID, tenant.TenantID)
		}
	}

	return nil
}

// Help function to determine if this aggregate has only the 'availability_zone' metadata
// attribute set. If so, the only purpose of the aggregate is to set the AZ for its member hosts.
func isAzAggregate(a aggregates.Aggregate) bool {
	if len(a.Metadata) == 1 {
		if _, ok := a.Metadata["availability_zone"]; ok {
			return true
		}
	}
	return false
}

func aggregatesLabel(h string, hostToAggrMap map[string][]string) string {
	if aggregates, ok := hostToAggrMap[h]; ok {
		slices.Sort(aggregates)
		return strings.Join(aggregates, ",")
	}
	return ""
}

// flavorIDMapper helper storage to map from Flavor Name to ID
type flavorIDMapper map[string]string

func newFlavorIDMapper(ctx context.Context, cli *gophercloud.ServiceClient) (flavorIDMapper, error) {
	allPagesFlavors, err := flavors.ListDetail(cli, flavors.ListOpts{AccessType: "None"}).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	allFlavors, err := flavors.ExtractFlavors(allPagesFlavors)
	if err != nil {
		return nil, err
	}

	m := make(flavorIDMapper, len(allFlavors))
	for _, f := range allFlavors {
		m[f.Name] = f.ID
	}

	return m, nil
}

func (s flavorIDMapper) Search(flavorName any) string {
	// flavor name is unique, making it suitable as the key for searching
	key, ok := flavorName.(string)
	if !ok {
		return ""
	}

	return s[key]
}
