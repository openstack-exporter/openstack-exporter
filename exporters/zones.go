package exporters

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/hypervisors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/services"
)

// computeHostZones includes the default compute zone, which is not necessarily
// represented by a host aggregate. Ignore scheduler/conductor services: their
// internal zone is not the zone of the host's compute resources.
func computeHostZones(ctx context.Context, client *gophercloud.ServiceClient) (map[string]string, error) {
	pages, err := services.List(client, services.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	allServices, err := services.ExtractServices(pages)
	if err != nil {
		return nil, err
	}
	zones := make(map[string]string)
	for _, service := range allServices {
		if service.Binary == "nova-compute" && service.Host != "" && service.Zone != "" {
			zones[service.Host] = service.Zone
		}
	}
	return zones, nil
}

// computeResourceZones uses the current exporter's credentials, region and
// endpoint interface, including when the compute exporter itself is disabled.
// Zone enrichment is optional: a failed Nova lookup must not drop resources
// collected successfully from Placement.
func computeResourceZones(ctx context.Context, exporter *BaseOpenStackExporter) map[string]string {
	zones, err := lookupComputeResourceZones(ctx, exporter)
	if err != nil {
		exporter.logger.Warn("Could not resolve compute resource zones", "exporter", exporter.Name, "err", err)
	}
	return zones
}

func lookupComputeResourceZones(ctx context.Context, exporter *BaseOpenStackExporter) (map[string]string, error) {
	if exporter.ClientV2.ProviderClient == nil || exporter.ClientV2.EndpointLocator == nil {
		return nil, fmt.Errorf("compute endpoint discovery is unavailable")
	}
	client, err := openstack.NewComputeV2(exporter.ClientV2.ProviderClient, exporter.EndpointOpts)
	if err != nil {
		return nil, err
	}
	zones, err := computeHostZones(ctx, client)
	if err != nil {
		return nil, err
	}
	pages, err := hypervisors.List(client, hypervisors.ListOpts{}).AllPages(ctx)
	if err != nil {
		return zones, err
	}
	allHypervisors, err := hypervisors.ExtractHypervisors(pages)
	if err != nil {
		return zones, err
	}
	// service.host and hypervisor_hostname may differ. The latter is also the
	// resource provider name, or the Ironic node UUID for bare metal hosts.
	for _, hypervisor := range allHypervisors {
		zone := zones[hypervisor.Service.Host]
		if zone == "" {
			continue
		}
		if hypervisor.HypervisorHostname != "" {
			zones[hypervisor.HypervisorHostname] = zone
		}
		if hypervisor.ID != "" {
			zones[hypervisor.ID] = zone
		}
	}
	return zones, nil
}
