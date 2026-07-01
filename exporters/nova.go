package exporters

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"

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
	identityprojects "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

func init() {
	RegisterTypedExporter("compute", NewNovaExporter)
}

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
	sched            Schedule
	descs            novaDescs
	serverStatusDesc *prometheus.Desc // dynamic labels (base + NovaMetadataMapping)
}

type novaDescs struct {
	Flavors                    *prometheus.Desc `metric:"flavors"`
	Flavor                     *prometheus.Desc `metric:"flavor"                        labels:"id,name,vcpus,ram,disk,is_public"`
	AvailabilityZones          *prometheus.Desc `metric:"availability_zones"`
	SecurityGroups             *prometheus.Desc `metric:"security_groups"`
	TotalVMs                   *prometheus.Desc `metric:"total_vms"`
	AgentState                 *prometheus.Desc `metric:"agent_state"                   labels:"id,hostname,service,adminState,zone,disabledReason"`
	RunningVMs                 *prometheus.Desc `metric:"running_vms"                   labels:"hostname,availability_zone,aggregates"`
	CurrentWorkload            *prometheus.Desc `metric:"current_workload"              labels:"hostname,availability_zone,aggregates"`
	VcpusAvailable             *prometheus.Desc `metric:"vcpus_available"               labels:"hostname,availability_zone,aggregates"`
	VcpusUsed                  *prometheus.Desc `metric:"vcpus_used"                    labels:"hostname,availability_zone,aggregates"`
	MemoryAvailableBytes       *prometheus.Desc `metric:"memory_available_bytes"        labels:"hostname,availability_zone,aggregates"`
	MemoryUsedBytes            *prometheus.Desc `metric:"memory_used_bytes"             labels:"hostname,availability_zone,aggregates"`
	LocalStorageAvailableBytes *prometheus.Desc `metric:"local_storage_available_bytes" labels:"hostname,availability_zone,aggregates"`
	LocalStorageUsedBytes      *prometheus.Desc `metric:"local_storage_used_bytes"      labels:"hostname,availability_zone,aggregates"`
	FreeDiskBytes              *prometheus.Desc `metric:"free_disk_bytes"               labels:"hostname,availability_zone,aggregates"`
	// server_status: stored directly in NovaExporter.serverStatusDesc
	LimitsVcpusMax                *prometheus.Desc `metric:"limits_vcpus_max"              labels:"tenant,tenant_id" slow:"true"`
	LimitsVcpusUsed               *prometheus.Desc `metric:"limits_vcpus_used"             labels:"tenant,tenant_id" slow:"true"`
	LimitsMemoryMax               *prometheus.Desc `metric:"limits_memory_max"             labels:"tenant,tenant_id" slow:"true"`
	LimitsMemoryUsed              *prometheus.Desc `metric:"limits_memory_used"            labels:"tenant,tenant_id" slow:"true"`
	LimitsInstancesUsed           *prometheus.Desc `metric:"limits_instances_used"         labels:"tenant,tenant_id" slow:"true"`
	LimitsInstancesMax            *prometheus.Desc `metric:"limits_instances_max"          labels:"tenant,tenant_id" slow:"true"`
	ServerLocalGB                 *prometheus.Desc `metric:"server_local_gb"               labels:"name,id,tenant_id" slow:"true"`
	QuotaCores                    *prometheus.Desc `metric:"quota_cores"                   labels:"type,tenant,tenant_id"`
	QuotaInstances                *prometheus.Desc `metric:"quota_instances"               labels:"type,tenant,tenant_id"`
	QuotaKeyPairs                 *prometheus.Desc `metric:"quota_key_pairs"               labels:"type,tenant,tenant_id"`
	QuotaMetadataItems            *prometheus.Desc `metric:"quota_metadata_items"          labels:"type,tenant,tenant_id"`
	QuotaRAM                      *prometheus.Desc `metric:"quota_ram"                     labels:"type,tenant,tenant_id"`
	QuotaServerGroups             *prometheus.Desc `metric:"quota_server_groups"           labels:"type,tenant,tenant_id"`
	QuotaServerGroupMembers       *prometheus.Desc `metric:"quota_server_group_members"    labels:"type,tenant,tenant_id"`
	QuotaFixedIPs                 *prometheus.Desc `metric:"quota_fixed_ips"               labels:"type,tenant,tenant_id"`
	QuotaFloatingIPs              *prometheus.Desc `metric:"quota_floating_ips"            labels:"type,tenant,tenant_id"`
	QuotaSecurityGroupRules       *prometheus.Desc `metric:"quota_security_group_rules"    labels:"type,tenant,tenant_id"`
	QuotaSecurityGroups           *prometheus.Desc `metric:"quota_security_groups"         labels:"type,tenant,tenant_id"`
	QuotaInjectedFileContentBytes *prometheus.Desc `metric:"quota_injected_file_content_bytes" labels:"type,tenant,tenant_id"`
	QuotaInjectedFilePathBytes    *prometheus.Desc `metric:"quota_injected_file_path_bytes"    labels:"type,tenant,tenant_id"`
	QuotaInjectedFiles            *prometheus.Desc `metric:"quota_injected_files"          labels:"type,tenant,tenant_id"`
}

type novaScrape struct {
	allFlavors     []flavors.Flavor
	allAZs         []availabilityzones.AvailabilityZone
	securityGroups []secgroups.SecurityGroup
	allServers     []servers.Server
	allServices    []services.Service
	allHypervisors []hypervisors.Hypervisor
	allAggregates  []aggregates.Aggregate
	projects       []identityprojects.Project
	limits         []novaProjectLimits
	tenantUsages   []usage.TenantUsage
	quotas         []novaProjectQuotas
}

type novaProjectLimits struct {
	projectName string
	projectID   string
	limits      limits.Limits
}

type novaProjectQuotas struct {
	projectName string
	projectID   string
	quota       quotasets.QuotaDetailSet
}

var novaGraph = Graph[*NovaExporter, novaScrape]{
	Sources: []Source[*NovaExporter, novaScrape]{
		{Name: "flavors", Fetch: (*NovaExporter).fetchFlavors},
		{Name: "azs", Fetch: (*NovaExporter).fetchAZs},
		{Name: "secgroups", Fetch: (*NovaExporter).fetchSecGroups},
		{Name: "servers", Fetch: (*NovaExporter).fetchServers},
		{Name: "services", Fetch: (*NovaExporter).fetchServices},
		{Name: "hypervisors", Fetch: (*NovaExporter).fetchHypervisors},
		{Name: "aggregates", Fetch: (*NovaExporter).fetchAggregates},
		{Name: "projects", Fetch: (*NovaExporter).fetchProjects},
		{Name: "limits", DependsOn: []string{"projects"}, Fetch: (*NovaExporter).fetchLimits},
		{Name: "usage", Fetch: (*NovaExporter).fetchUsage},
		{Name: "quotas", DependsOn: []string{"projects"}, Fetch: (*NovaExporter).fetchQuotas},
	},
	Emitters: []Emitter[*NovaExporter, novaScrape]{
		{Name: "flavors", Metrics: []string{"flavors", "flavor"}, Sources: []string{"flavors"}, Emit: (*NovaExporter).emitFlavors},
		{Name: "azs", Metrics: []string{"availability_zones"}, Sources: []string{"azs"}, Emit: (*NovaExporter).emitAZs},
		{Name: "secgroups", Metrics: []string{"security_groups"}, Sources: []string{"secgroups"}, Emit: (*NovaExporter).emitSecGroups},
		{Name: "server_count", Metrics: []string{"total_vms"}, Sources: []string{"servers"}, Emit: (*NovaExporter).emitServerCount},
		{Name: "server_status", Metrics: []string{"server_status"}, Sources: []string{"servers", "flavors"}, Emit: (*NovaExporter).emitServerStatus},
		{Name: "services", Metrics: []string{"agent_state"}, Sources: []string{"services"}, Emit: (*NovaExporter).emitServices},
		{Name: "hypervisors", Metrics: []string{"running_vms", "current_workload", "vcpus_available", "vcpus_used", "memory_available_bytes", "memory_used_bytes", "local_storage_available_bytes", "local_storage_used_bytes", "free_disk_bytes"}, Sources: []string{"hypervisors", "aggregates"}, Emit: (*NovaExporter).emitHypervisors},
		{Name: "limits", Metrics: []string{"limits_vcpus_max", "limits_vcpus_used", "limits_memory_max", "limits_memory_used", "limits_instances_used", "limits_instances_max"}, Sources: []string{"limits"}, Emit: (*NovaExporter).emitLimits},
		{Name: "usage", Metrics: []string{"server_local_gb"}, Sources: []string{"usage"}, Emit: (*NovaExporter).emitUsage},
		{Name: "quotas", Metrics: []string{"quota_cores", "quota_instances", "quota_key_pairs", "quota_metadata_items", "quota_ram", "quota_server_groups", "quota_server_group_members", "quota_fixed_ips", "quota_floating_ips", "quota_security_group_rules", "quota_security_groups", "quota_injected_file_content_bytes", "quota_injected_file_path_bytes", "quota_injected_files"}, Sources: []string{"quotas"}, Emit: (*NovaExporter).emitQuotas},
	},
}

func NewNovaExporter(config *ExporterConfig, logger *slog.Logger) (*NovaExporter, error) {
	ctx := context.TODO()
	if err := utils.SetupClientMicroversionV2(ctx, config.ClientV2, "OS_COMPUTE_API_VERSION", novaLatestSupportedMicroversion, logger); err != nil {
		return nil, err
	}
	e := &NovaExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "nova",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	e.RegisterAndFillDescs(&e.descs)
	// server_status has dynamic extra labels from NovaMetadataMapping — create directly
	serverStatusBaseLabels := []string{"id", "status", "name", "tenant_id", "user_id", "address_ipv4",
		"address_ipv6", "host_id", "hypervisor_hostname", "uuid", "availability_zone", "flavor_id", "instance_libvirt"}
	allLabels := append(serverStatusBaseLabels, config.NovaMetadataMapping.Labels...)
	if e.IsMetricEnabled("server_status") {
		e.serverStatusDesc = prometheus.NewDesc(
			prometheus.BuildFQName(e.GetName(), "", "server_status"),
			"server_status", allLabels, nil)
		e.RegisterDesc(e.serverStatusDesc)
	}

	sched, err := novaGraph.PruneSchedule(&e.BaseOpenStackExporter)
	if err != nil {
		return nil, err
	}
	e.sched = sched
	novaGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)
	return e, nil
}

func (e *NovaExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(novaScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &novaGraph, e.sched, s, ch)
	})
}

// --- Sources ---

func (e *NovaExporter) fetchFlavors(ctx context.Context, s *novaScrape) error {
	allPages, err := flavors.ListDetail(e.ClientV2, flavors.ListOpts{AccessType: "None"}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.allFlavors, err = flavors.ExtractFlavors(allPages)
	return err
}

func (e *NovaExporter) fetchAZs(ctx context.Context, s *novaScrape) error {
	allPages, err := availabilityzones.List(e.ClientV2).AllPages(ctx)
	if err != nil {
		return err
	}
	s.allAZs, err = availabilityzones.ExtractAvailabilityZones(allPages)
	return err
}

func (e *NovaExporter) fetchSecGroups(ctx context.Context, s *novaScrape) error {
	allPages, err := secgroups.List(e.ClientV2).AllPages(ctx)
	if err != nil {
		return err
	}
	s.securityGroups, err = secgroups.ExtractSecurityGroups(allPages)
	return err
}

func (e *NovaExporter) fetchServers(ctx context.Context, s *novaScrape) error {
	opts := getServerListOptions(e.TenantID)
	allPages, err := servers.List(e.ClientV2, opts).AllPages(ctx)
	if err != nil {
		return err
	}
	if err := servers.ExtractServersInto(allPages, &s.allServers); err != nil {
		return err
	}
	return nil
}

func (e *NovaExporter) fetchServices(ctx context.Context, s *novaScrape) error {
	allPages, err := services.List(e.ClientV2, services.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.allServices, err = services.ExtractServices(allPages)
	return err
}

func (e *NovaExporter) fetchHypervisors(ctx context.Context, s *novaScrape) error {
	var listOpts *hypervisors.ListOpts
	if ok, _ := utils.IsMicroversionAtLeast(e.ClientV2.Microversion, "2.33"); ok {
		listOpts = &hypervisors.ListOpts{Limit: new(1000)}
	} else {
		listOpts = &hypervisors.ListOpts{}
	}
	allPages, err := hypervisors.List(e.ClientV2, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	s.allHypervisors, err = hypervisors.ExtractHypervisors(allPages)
	if err != nil {
		return err
	}
	return nil
}

func (e *NovaExporter) fetchAggregates(ctx context.Context, s *novaScrape) error {
	allPagesAggr, err := aggregates.List(e.ClientV2).AllPages(ctx)
	if err != nil {
		return err
	}
	s.allAggregates, err = aggregates.ExtractAggregates(allPagesAggr)
	return err
}

func (e *NovaExporter) fetchProjects(ctx context.Context, s *novaScrape) error {
	var err error
	s.projects, err = GetProjects(ctx, &e.BaseOpenStackExporter)
	return err
}

func (e *NovaExporter) fetchLimits(ctx context.Context, s *novaScrape) error {
	s.limits = make([]novaProjectLimits, len(s.projects))
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(e.GetAPIDetailConcurrencyCount())
	for i, p := range s.projects {
		i, p := i, p
		opts := limits.GetOpts{TenantID: p.ID}
		if p.ID == e.TenantID {
			opts = limits.GetOpts{}
		}
		g.Go(func() error {
			lim, err := limits.Get(gCtx, e.ClientV2, opts).Extract()
			if err != nil {
				return err
			}
			s.limits[i] = novaProjectLimits{
				projectName: p.Name,
				projectID:   p.ID,
				limits:      *lim,
			}
			return nil
		})
	}
	return g.Wait()
}

func (e *NovaExporter) fetchUsage(ctx context.Context, s *novaScrape) error {
	if e.descs.ServerLocalGB == nil {
		return nil
	}
	allPages, err := usage.AllTenants(e.ClientV2, usage.AllTenantsOpts{Detailed: true}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.tenantUsages, err = usage.ExtractAllTenants(allPages)
	return err
}

func (e *NovaExporter) fetchQuotas(ctx context.Context, s *novaScrape) error {
	s.quotas = make([]novaProjectQuotas, len(s.projects))
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(e.GetAPIDetailConcurrencyCount())
	for i, p := range s.projects {
		i, p := i, p
		g.Go(func() error {
			quotaSet, err := quotasets.GetDetail(gCtx, e.ClientV2, p.ID).Extract()
			if err != nil {
				return err
			}
			s.quotas[i] = novaProjectQuotas{
				projectName: p.Name,
				projectID:   p.ID,
				quota:       quotaSet,
			}
			return nil
		})
	}
	return g.Wait()
}

// --- Emitters ---

func (e *NovaExporter) emitFlavors(ctx context.Context, s *novaScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.Flavors, float64(len(s.allFlavors)))
	for _, f := range s.allFlavors {
		emitGauge(ch, e.descs.Flavor, 1, f.ID, f.Name, fmt.Sprintf("%v", f.VCPUs), fmt.Sprintf("%v", f.RAM),
			fmt.Sprintf("%v", f.Disk), fmt.Sprintf("%v", f.IsPublic))
	}
	return nil
}

func (e *NovaExporter) emitAZs(ctx context.Context, s *novaScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.AvailabilityZones, float64(len(s.allAZs)))
	return nil
}

func (e *NovaExporter) emitSecGroups(ctx context.Context, s *novaScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.SecurityGroups, float64(len(s.securityGroups)))
	return nil
}

func (e *NovaExporter) emitServerCount(ctx context.Context, s *novaScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.TotalVMs, float64(len(s.allServers)))
	return nil
}

func (e *NovaExporter) emitServerStatus(ctx context.Context, s *novaScrape, ch chan<- prometheus.Metric) error {
	var flavorMapper flavorIDMapper
	mvAtLeast246, _ := utils.IsMicroversionAtLeast(e.ClientV2.Microversion, "2.46")
	if mvAtLeast246 || serversNeedFlavorMapper(s.allServers) {
		flavorMapper = newFlavorIDMapper(s.allFlavors)
	}
	for _, server := range s.allServers {
		var flavorID string
		if flavorMapper == nil {
			flavorID = fmt.Sprintf("%v", server.Flavor["id"])
		} else {
			flavorID = flavorMapper.Search(server.Flavor["original_name"])
		}
		labelValues := []string{
			server.ID, server.Status, server.Name, server.TenantID,
			server.UserID, server.AccessIPv4, server.AccessIPv6, server.HostID,
			server.HypervisorHostname, server.ID,
			server.AvailabilityZone, flavorID, server.InstanceName,
		}
		metadataValues := e.NovaMetadataMapping.Extract(server.Metadata)
		emitGauge(ch, e.serverStatusDesc,
			float64(mapServerStatus(server.Status)), append(labelValues, metadataValues...)...)
	}
	return nil
}

func (e *NovaExporter) emitServices(ctx context.Context, s *novaScrape, ch chan<- prometheus.Metric) error {
	for _, svc := range s.allServices {
		state := 0
		if svc.State == "up" {
			state = 1
		}
		emitGauge(ch, e.descs.AgentState,
			float64(state), svc.ID, svc.Host, svc.Binary, svc.Status, svc.Zone, svc.DisabledReason)
	}
	return nil
}

func (e *NovaExporter) emitHypervisors(ctx context.Context, s *novaScrape, ch chan<- prometheus.Metric) error {
	hostToAzMap := map[string]string{}
	hostToAggrMap := map[string][]string{}
	for _, a := range s.allAggregates {
		isAz := isAzAggregate(a)
		for _, h := range a.Hosts {
			if a.AvailabilityZone != "" {
				hostToAzMap[h] = a.AvailabilityZone
			}
			if !isAz {
				hostToAggrMap[h] = append(hostToAggrMap[h], a.Name)
			}
		}
	}
	for _, hv := range s.allHypervisors {
		az := hostToAzMap[hv.Service.Host]
		aggr := aggregatesLabel(hv.Service.Host, hostToAggrMap)
		var vcpus int
		if !reflect.ValueOf(hv.CPUInfo).IsZero() {
			vcpus = max(hv.CPUInfo.Topology.Cells, 1) * hv.CPUInfo.Topology.Sockets * hv.CPUInfo.Topology.Cores * hv.CPUInfo.Topology.Threads
		} else {
			vcpus = hv.VCPUs
		}
		emit := func(d *prometheus.Desc, v float64) {
			emitGauge(ch, d, v, hv.HypervisorHostname, az, aggr)
		}
		emit(e.descs.RunningVMs, float64(hv.RunningVMs))
		emit(e.descs.CurrentWorkload, float64(hv.CurrentWorkload))
		emit(e.descs.VcpusAvailable, float64(vcpus))
		emit(e.descs.VcpusUsed, float64(hv.VCPUsUsed))
		emit(e.descs.MemoryAvailableBytes, float64(hv.MemoryMB*MEGABYTE))
		emit(e.descs.MemoryUsedBytes, float64(hv.MemoryMBUsed*MEGABYTE))
		emit(e.descs.LocalStorageAvailableBytes, float64(hv.LocalGB*GIGABYTE))
		emit(e.descs.LocalStorageUsedBytes, float64(hv.LocalGBUsed*GIGABYTE))
		emit(e.descs.FreeDiskBytes, float64(hv.FreeDiskGB*GIGABYTE))
	}
	return nil
}

func (e *NovaExporter) emitLimits(ctx context.Context, s *novaScrape, ch chan<- prometheus.Metric) error {
	for _, lim := range s.limits {
		absolute := lim.limits.Absolute
		emit := func(d *prometheus.Desc, v float64) {
			emitGauge(ch, d, v, lim.projectName, lim.projectID)
		}
		emit(e.descs.LimitsVcpusMax, float64(absolute.MaxTotalCores))
		emit(e.descs.LimitsVcpusUsed, float64(absolute.TotalCoresUsed))
		emit(e.descs.LimitsMemoryMax, float64(absolute.MaxTotalRAMSize))
		emit(e.descs.LimitsMemoryUsed, float64(absolute.TotalRAMUsed))
		emit(e.descs.LimitsInstancesUsed, float64(absolute.TotalInstancesUsed))
		emit(e.descs.LimitsInstancesMax, float64(absolute.MaxTotalInstances))
	}
	return nil
}

func (e *NovaExporter) emitUsage(ctx context.Context, s *novaScrape, ch chan<- prometheus.Metric) error {
	for _, tenant := range s.tenantUsages {
		for _, server := range tenant.ServerUsages {
			emitGauge(ch, e.descs.ServerLocalGB,
				float64(server.LocalGB), server.Name, server.InstanceID, tenant.TenantID)
		}
	}
	return nil
}

func (e *NovaExporter) emitQuotas(ctx context.Context, s *novaScrape, ch chan<- prometheus.Metric) error {
	for _, entry := range s.quotas {
		quotaSet := entry.quota
		e.emitNovaQuotaDetail(ch, e.descs.QuotaCores, quotaSet.Cores, entry.projectName, entry.projectID)
		e.emitNovaQuotaDetail(ch, e.descs.QuotaInstances, quotaSet.Instances, entry.projectName, entry.projectID)
		e.emitNovaQuotaDetail(ch, e.descs.QuotaKeyPairs, quotaSet.KeyPairs, entry.projectName, entry.projectID)
		e.emitNovaQuotaDetail(ch, e.descs.QuotaMetadataItems, quotaSet.MetadataItems, entry.projectName, entry.projectID)
		e.emitNovaQuotaDetail(ch, e.descs.QuotaRAM, quotaSet.RAM, entry.projectName, entry.projectID)
		e.emitNovaQuotaDetail(ch, e.descs.QuotaServerGroups, quotaSet.ServerGroups, entry.projectName, entry.projectID)
		e.emitNovaQuotaDetail(ch, e.descs.QuotaServerGroupMembers, quotaSet.ServerGroupMembers, entry.projectName, entry.projectID)
		e.emitNovaQuotaDetail(ch, e.descs.QuotaFixedIPs, quotaSet.FixedIPs, entry.projectName, entry.projectID)
		e.emitNovaQuotaDetail(ch, e.descs.QuotaFloatingIPs, quotaSet.FloatingIPs, entry.projectName, entry.projectID)
		e.emitNovaQuotaDetail(ch, e.descs.QuotaSecurityGroupRules, quotaSet.SecurityGroupRules, entry.projectName, entry.projectID)
		e.emitNovaQuotaDetail(ch, e.descs.QuotaSecurityGroups, quotaSet.SecurityGroups, entry.projectName, entry.projectID)
		e.emitNovaQuotaDetail(ch, e.descs.QuotaInjectedFileContentBytes, quotaSet.InjectedFileContentBytes, entry.projectName, entry.projectID)
		e.emitNovaQuotaDetail(ch, e.descs.QuotaInjectedFilePathBytes, quotaSet.InjectedFilePathBytes, entry.projectName, entry.projectID)
		e.emitNovaQuotaDetail(ch, e.descs.QuotaInjectedFiles, quotaSet.InjectedFiles, entry.projectName, entry.projectID)
	}
	return nil
}

func (e *NovaExporter) emitNovaQuotaDetail(ch chan<- prometheus.Metric, desc *prometheus.Desc, q quotasets.QuotaDetail, projectName, projectID string) {
	emitGauge(ch, desc, float64(q.InUse), "in_use", projectName, projectID)
	emitGauge(ch, desc, float64(q.Reserved), "reserved", projectName, projectID)
	emitGauge(ch, desc, float64(q.Limit), "limit", projectName, projectID)
}

func getServerListOptions(tenantID string) servers.ListOpts {
	if tenantID == "" {
		return servers.ListOpts{AllTenants: true}
	}
	return servers.ListOpts{TenantID: tenantID}
}

func isAzAggregate(a aggregates.Aggregate) bool {
	if len(a.Metadata) == 1 {
		if _, ok := a.Metadata["availability_zone"]; ok {
			return true
		}
	}
	return false
}

func aggregatesLabel(h string, hostToAggrMap map[string][]string) string {
	if aggs, ok := hostToAggrMap[h]; ok {
		slices.Sort(aggs)
		return strings.Join(aggs, ",")
	}
	return ""
}

type flavorIDMapper map[string]string

func newFlavorIDMapper(allFlavors []flavors.Flavor) flavorIDMapper {
	m := make(flavorIDMapper, len(allFlavors))
	for _, f := range allFlavors {
		m[f.Name] = f.ID
	}
	return m
}

func serversNeedFlavorMapper(allServers []servers.Server) bool {
	for _, srv := range allServers {
		if _, ok := srv.Flavor["id"]; !ok {
			return true
		}
	}
	return false
}

func (s flavorIDMapper) Search(flavorName any) string {
	key, ok := flavorName.(string)
	if !ok {
		return ""
	}
	return s[key]
}
