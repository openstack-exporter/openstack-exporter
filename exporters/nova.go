package exporters

import (
	"errors"
	"fmt"
	"sort"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/aggregates"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/hypervisors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/limits"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/services"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/prometheus/client_golang/prometheus"
)

var server_status = []string{
	"ACTIVE",
	"BUILD",             // The server has not finished the original build process.
	"BUILD(spawning)",   // The server has not finished the original build process but networking works (HP Cloud specific)
	"DELETED",           // The server is deleted.
	"ERROR",             // The server is in error.
	"HARD_REBOOT",       // The server is hard rebooting.
	"PASSWORD",          // The password is being reset on the server.
	"REBOOT",            // The server is in a soft reboot state.
	"REBUILD",           // The server is currently being rebuilt from an image.
	"RESCUE",            // The server is in rescue mode.
	"RESIZE",            // Server is performing the differential copy of data that changed during its initial copy.
	"SHUTOFF",           // The virtual machine (VM) was powered down by the user, but not through the OpenStack Compute API.
	"SUSPENDED",         // The server is suspended, either by request or necessity.
	"UNKNOWN",           // The state of the server is unknown. Contact your cloud provider.
	"VERIFY_RESIZE",     // System is awaiting confirmation that the server is operational after a move or resize.
	"MIGRATING",         // The server is migrating. This is caused by a live migration (moving a server that is active) action.
	"PAUSED",            // The server is paused.
	"REVERT_RESIZE",     // The resize or migration of a server failed for some reason. The destination server is being cleaned up and the original source server is restarting.
	"SHELVED",           // The server is in shelved state. Depends on the shelve offload time, the server will be automatically shelved off loaded.
	"SHELVED_OFFLOADED", // The shelved server is offloaded (removed from the compute host) and it needs unshelved action to be used again.
	"SOFT_DELETED",      // The server is marked as deleted but will remain in the cloud for some configurable amount of time.
}

func mapServerStatus(current string) int {
	for idx, status := range server_status {
		if current == status {
			return idx
		}
	}
	return -1
}

type NovaExporter struct {
	BaseOpenStackExporter
}

var defaultNovaMetrics = []Metric{
	{Name: "flavors", Fn: ListFlavors},
	{Name: "availability_zones", Fn: ListAZs},
	{Name: "security_groups", Fn: ListComputeSecGroups},
	{Name: "total_vms", Fn: ListAllServers},
	{Name: "agent_state", Labels: []string{"id", "hostname", "service", "adminState", "zone", "disabledReason"}, Fn: ListNovaAgentState},
	{Name: "running_vms", Labels: []string{"hostname", "availability_zone", "aggregates"}, Fn: ListHypervisors},
	{Name: "current_workload", Labels: []string{"hostname", "availability_zone", "aggregates"}},
	{Name: "vcpus_available", Labels: []string{"hostname", "availability_zone", "aggregates"}},
	{Name: "vcpus_used", Labels: []string{"hostname", "availability_zone", "aggregates"}},
	{Name: "memory_available_bytes", Labels: []string{"hostname", "availability_zone", "aggregates"}},
	{Name: "memory_used_bytes", Labels: []string{"hostname", "availability_zone", "aggregates"}},
	{Name: "local_storage_available_bytes", Labels: []string{"hostname", "availability_zone", "aggregates"}},
	{Name: "local_storage_used_bytes", Labels: []string{"hostname", "availability_zone", "aggregates"}},
	{Name: "server_status", Labels: []string{"id", "status", "name", "tenant_id", "user_id", "address_ipv4",
		"address_ipv6", "host_id", "uuid", "availability_zone", "flavor_id"}},
	{Name: "limits_vcpus_max", Labels: []string{"tenant", "tenant_id"}, Fn: ListComputeLimits},
	{Name: "limits_vcpus_used", Labels: []string{"tenant", "tenant_id"}},
	{Name: "limits_memory_max", Labels: []string{"tenant", "tenant_id"}},
	{Name: "limits_memory_used", Labels: []string{"tenant", "tenant_id"}},
}

func NewNovaExporter(config *ExporterConfig) (*NovaExporter, error) {
	exporter := NovaExporter{
		BaseOpenStackExporter{
			Name:           "nova",
			ExporterConfig: *config,
		},
	}
	for _, metric := range defaultNovaMetrics {
		exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, nil)
	}

	return &exporter, nil
}

func ListNovaAgentState(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allServices []services.Service

	allPagesServices, err := services.List(exporter.Client).AllPages()
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

func ListHypervisors(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allHypervisors []hypervisors.Hypervisor
	var allAggregates []aggregates.Aggregate

	allPagesHypervisors, err := hypervisors.List(exporter.Client).AllPages()
	if err != nil {
		return err
	}

	if allHypervisors, err = hypervisors.ExtractHypervisors(allPagesHypervisors); err != nil {
		return err
	}

	allPagesAggregates, err := aggregates.List(exporter.Client).AllPages()
	if err != nil {
		return err
	}

	if allAggregates, err = aggregates.ExtractAggregates(allPagesAggregates); err != nil {
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

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["running_vms"].Metric,
			prometheus.GaugeValue, float64(hypervisor.RunningVMs), hypervisor.HypervisorHostname, availabilityZone, aggregatesLabel(hypervisor.Service.Host, hostToAggrMap))

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["current_workload"].Metric,
			prometheus.GaugeValue, float64(hypervisor.CurrentWorkload), hypervisor.HypervisorHostname, availabilityZone, aggregatesLabel(hypervisor.Service.Host, hostToAggrMap))

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["vcpus_available"].Metric,
			prometheus.GaugeValue, float64(hypervisor.VCPUs), hypervisor.HypervisorHostname, availabilityZone, aggregatesLabel(hypervisor.Service.Host, hostToAggrMap))

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["vcpus_used"].Metric,
			prometheus.GaugeValue, float64(hypervisor.VCPUsUsed), hypervisor.HypervisorHostname, availabilityZone, aggregatesLabel(hypervisor.Service.Host, hostToAggrMap))

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["memory_available_bytes"].Metric,
			prometheus.GaugeValue, float64(hypervisor.MemoryMB*MEGABYTE), hypervisor.HypervisorHostname, availabilityZone, aggregatesLabel(hypervisor.Service.Host, hostToAggrMap))

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["memory_used_bytes"].Metric,
			prometheus.GaugeValue, float64(hypervisor.MemoryMBUsed*MEGABYTE), hypervisor.HypervisorHostname, availabilityZone, aggregatesLabel(hypervisor.Service.Host, hostToAggrMap))

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["local_storage_available_bytes"].Metric,
			prometheus.GaugeValue, float64(hypervisor.LocalGB*GIGABYTE), hypervisor.HypervisorHostname, availabilityZone, aggregatesLabel(hypervisor.Service.Host, hostToAggrMap))

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["local_storage_used_bytes"].Metric,
			prometheus.GaugeValue, float64(hypervisor.LocalGBUsed*GIGABYTE), hypervisor.HypervisorHostname, availabilityZone, aggregatesLabel(hypervisor.Service.Host, hostToAggrMap))
	}

	return nil
}

func ListFlavors(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allFlavors []flavors.Flavor

	allPagesFlavors, err := flavors.ListDetail(exporter.Client, flavors.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allFlavors, err = flavors.ExtractFlavors(allPagesFlavors)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["flavors"].Metric,
		prometheus.GaugeValue, float64(len(allFlavors)))

	return nil
}

func ListAZs(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allAZs []availabilityzones.AvailabilityZone

	allPagesAZs, err := availabilityzones.List(exporter.Client).AllPages()
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

func ListComputeSecGroups(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allSecurityGroups []secgroups.SecurityGroup

	allPagesSecurityGroups, err := secgroups.List(exporter.Client).AllPages()
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

func ListAllServers(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	type ServerWithExt struct {
		servers.Server
		availabilityzones.ServerAvailabilityZoneExt
	}

	var allServers []ServerWithExt

	allPagesServers, err := servers.List(exporter.Client, servers.ListOpts{AllTenants: true}).AllPages()
	if err != nil {
		return err
	}

	err = servers.ExtractServersInto(allPagesServers, &allServers)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["total_vms"].Metric,
		prometheus.GaugeValue, float64(len(allServers)))

	// Server status metrics
	for _, server := range allServers {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["server_status"].Metric,
			prometheus.GaugeValue, float64(mapServerStatus(server.Status)), server.ID, server.Status, server.Name, server.TenantID,
			server.UserID, server.AccessIPv4, server.AccessIPv6, server.HostID, server.ID, server.AvailabilityZone, fmt.Sprintf("%v", server.Flavor["id"]))
	}

	return nil
}

func ListComputeLimits(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allProjects []projects.Project
	var eo gophercloud.EndpointOpts

	// We need a list of all tenants/projects. Therefore, within this nova exporter we need
	// to create an openstack client for the Identity/Keystone API.
	// If possible, use the EndpointOpts spefic to the identity service.
	if v, ok := endpointOpts["identity"]; ok {
		eo = v
	} else if v, ok := endpointOpts["compute"]; ok {
		eo = v
	} else {
		return errors.New("No EndpointOpts available to create Identity client")
	}

	c, err := openstack.NewIdentityV3(exporter.Client.ProviderClient, eo)
	if err != nil {
		return err
	}

	allPagesProject, err := projects.List(c, projects.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allProjects, err = projects.ExtractProjects(allPagesProject)
	if err != nil {
		return err
	}

	for _, p := range allProjects {
		// Limits are obtained from the nova API, so now we can just use this exporter's client
		limits, err := limits.Get(exporter.Client, limits.GetOpts{TenantID: p.ID}).Extract()
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
	label := ""
	if aggregates, ok := hostToAggrMap[h]; ok {
		sort.Strings(aggregates)
		for k, a := range aggregates {
			if k == 0 {
				label += a
			} else {
				label += "," + a
			}
		}
	}
	return label
}
