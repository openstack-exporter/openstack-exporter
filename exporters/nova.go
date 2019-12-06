package exporters

import (
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/aggregates"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/hypervisors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/services"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
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
	{Name: "agent_state", Labels: []string{"id", "hostname", "service", "adminState", "zone"}, Fn: ListNovaAgentState},
	{Name: "running_vms", Labels: []string{"hostname", "availability_zone"}, Fn: ListHypervisors},
	{Name: "current_workload", Labels: []string{"hostname", "availability_zone"}},
	{Name: "vcpus_available", Labels: []string{"hostname", "availability_zone"}},
	{Name: "vcpus_used", Labels: []string{"hostname", "availability_zone"}},
	{Name: "memory_available_bytes", Labels: []string{"hostname", "availability_zone"}},
	{Name: "memory_used_bytes", Labels: []string{"hostname", "availability_zone"}},
	{Name: "local_storage_available_bytes", Labels: []string{"hostname", "availability_zone"}},
	{Name: "local_storage_used_bytes", Labels: []string{"hostname", "availability_zone"}},
	{Name: "server_status", Labels: []string{"id", "status", "name", "tenant_id", "user_id", "address_ipv4",
		"address_ipv6", "host_id", "uuid", "availability_zone", "flavor_id"}},
}

func NewNovaExporter(client *gophercloud.ServiceClient, prefix string, disabledMetrics []string) (*NovaExporter, error) {
	exporter := NovaExporter{
		BaseOpenStackExporter{
			Name:            "nova",
			Prefix:          prefix,
			Client:          client,
			DisabledMetrics: disabledMetrics,
		},
	}
	for _, metric := range defaultNovaMetrics {
		exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, nil)
	}

	return &exporter, nil
}

func (exporter *NovaExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range exporter.Metrics {
		ch <- metric.Metric
	}
}

func (exporter *NovaExporter) Collect(ch chan<- prometheus.Metric) {
	exporter.CollectMetrics(ch)
}

func ListNovaAgentState(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching list of services")
	var allServices []services.Service

	allPagesServices, err := services.List(exporter.Client).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	if allServices, err = services.ExtractServices(allPagesServices); err != nil {
		log.Errorln(err)
		return
	}

	for _, service := range allServices {
		var state = 0
		if service.State == "up" {
			state = 1
		}
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["agent_state"].Metric,
			prometheus.CounterValue, float64(state), service.ID, service.Host, service.Binary, service.Status, service.Zone)
	}
}

func ListHypervisors(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching list of hypervisors and aggregates")
	var allHypervisors []hypervisors.Hypervisor
	var allAggregates []aggregates.Aggregate

	allPagesHypervisors, err := hypervisors.List(exporter.Client).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	if allHypervisors, err = hypervisors.ExtractHypervisors(allPagesHypervisors); err != nil {
		log.Errorln(err)
		return
	}

	allPagesAggregates, err := aggregates.List(exporter.Client).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	if allAggregates, err = aggregates.ExtractAggregates(allPagesAggregates); err != nil {
		log.Errorln(err)
		return
	}

	hostToAggregateMap := map[string]string{}
	for _, v := range allAggregates {
		for _, w := range v.Hosts {
			if v.AvailabilityZone != "" {
				hostToAggregateMap[w] = v.AvailabilityZone
			}
		}
	}

	for _, hypervisor := range allHypervisors {
		availabilityZone := ""
		if val, ok := hostToAggregateMap[hypervisor.HypervisorHostname]; ok {
			availabilityZone = val
		}

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["running_vms"].Metric,
			prometheus.GaugeValue, float64(hypervisor.RunningVMs), hypervisor.HypervisorHostname, availabilityZone)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["current_workload"].Metric,
			prometheus.GaugeValue, float64(hypervisor.CurrentWorkload), hypervisor.HypervisorHostname, availabilityZone)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["vcpus_available"].Metric,
			prometheus.GaugeValue, float64(hypervisor.VCPUs), hypervisor.HypervisorHostname, availabilityZone)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["vcpus_used"].Metric,
			prometheus.GaugeValue, float64(hypervisor.VCPUsUsed), hypervisor.HypervisorHostname, availabilityZone)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["memory_available_bytes"].Metric,
			prometheus.GaugeValue, float64(hypervisor.MemoryMB*MEGABYTE), hypervisor.HypervisorHostname, availabilityZone)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["memory_used_bytes"].Metric,
			prometheus.GaugeValue, float64(hypervisor.MemoryMBUsed*MEGABYTE), hypervisor.HypervisorHostname, availabilityZone)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["local_storage_available_bytes"].Metric,
			prometheus.GaugeValue, float64(hypervisor.LocalGB*GIGABYTE), hypervisor.HypervisorHostname, availabilityZone)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["local_storage_used_bytes"].Metric,
			prometheus.GaugeValue, float64(hypervisor.LocalGBUsed*GIGABYTE), hypervisor.HypervisorHostname, availabilityZone)
	}
}

func ListFlavors(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching list of flavors")
	var allFlavors []flavors.Flavor

	allPagesFlavors, err := flavors.ListDetail(exporter.Client, flavors.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allFlavors, err = flavors.ExtractFlavors(allPagesFlavors)
	if err != nil {
		log.Errorln(err)
		return
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["flavors"].Metric,
		prometheus.GaugeValue, float64(len(allFlavors)))
}

func ListAZs(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching list of availability zones")
	var allAZs []availabilityzones.AvailabilityZone

	allPagesAZs, err := availabilityzones.List(exporter.Client).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	if allAZs, err = availabilityzones.ExtractAvailabilityZones(allPagesAZs); err != nil {
		log.Errorln(err)
		return
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["availability_zones"].Metric,
		prometheus.GaugeValue, float64(len(allAZs)))
}

func ListComputeSecGroups(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching list of nova security groups")
	var allSecurityGroups []secgroups.SecurityGroup

	allPagesSecurityGroups, err := secgroups.List(exporter.Client).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	if allSecurityGroups, err = secgroups.ExtractSecurityGroups(allPagesSecurityGroups); err != nil {
		log.Errorln(err)
		return
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["security_groups"].Metric,
		prometheus.GaugeValue, float64(len(allSecurityGroups)))
}

func ListAllServers(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching list of servers for all tenants")

	type ServerWithExt struct {
		servers.Server
		availabilityzones.ServerAvailabilityZoneExt
	}

	var allServers []ServerWithExt

	allPagesServers, err := servers.List(exporter.Client, servers.ListOpts{AllTenants: true}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	err = servers.ExtractServersInto(allPagesServers, &allServers)
	if err != nil {
		log.Errorln(err)
		return
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["total_vms"].Metric,
		prometheus.GaugeValue, float64(len(allServers)))

	// Server status metrics
	for _, server := range allServers {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["server_status"].Metric,
			prometheus.GaugeValue, float64(mapServerStatus(server.Status)), server.ID, server.Status, server.Name, server.TenantID,
			server.UserID, server.AccessIPv4, server.AccessIPv6, server.HostID, server.ID, server.AvailabilityZone, fmt.Sprintf("%v", server.Flavor["id"]))
	}
}
