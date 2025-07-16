package exporters

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-kit/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/quotasets"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/schedulerstats"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/services"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/volumetenants"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/prometheus/client_golang/prometheus"
)

type CinderExporter struct {
	BaseOpenStackExporter
}

var volume_status = []string{
	"creating",
	"available",
	"reserved",
	"attaching",
	"detaching",
	"in-use",
	"maintenance",
	"deleting",
	"awaiting-transfer",
	"error",
	"error_deleting",
	"backing-up",
	"restoring-backup",
	"error_backing-up",
	"error_restoring",
	"error_extending",
	"downloading",
	"uploading",
	"retyping",
	"extending",
}

func mapVolumeStatus(volStatus string) int {
	for idx, status := range volume_status {
		if status == strings.ToLower(volStatus) {
			return idx
		}
	}
	return -1
}

var defaultCinderMetrics = []Metric{
	{Name: "volumes", Fn: ListVolumes},
	{Name: "snapshots", Fn: ListSnapshots},
	{Name: "agent_state", Labels: []string{"uuid", "hostname", "service", "adminState", "zone", "disabledReason"}, Fn: ListCinderAgentState},
	{Name: "volume_gb", Labels: []string{"id", "name", "status", "availability_zone", "bootable", "tenant_id", "user_id", "volume_type", "server_id"}, Fn: nil},
	{Name: "volume_status", Labels: []string{"id", "name", "status", "bootable", "tenant_id", "size", "volume_type", "server_id"}, Fn: ListVolumesStatus, Slow: false, DeprecatedVersion: "1.4"},
	{Name: "volume_status_counter", Labels: []string{"status"}, Fn: nil},
	{Name: "pool_capacity_free_gb", Labels: []string{"name", "volume_backend_name", "vendor_name"}, Fn: ListCinderPoolCapacityFree},
	{Name: "pool_capacity_total_gb", Labels: []string{"name", "volume_backend_name", "vendor_name"}, Fn: nil},
	{Name: "limits_volume_max_gb", Labels: []string{"tenant", "tenant_id"}, Fn: ListVolumeLimits, Slow: true},
	{Name: "limits_volume_used_gb", Labels: []string{"tenant", "tenant_id"}, Fn: nil, Slow: true},
	{Name: "limits_backup_max_gb", Labels: []string{"tenant", "tenant_id"}, Fn: nil, Slow: true},
	{Name: "limits_backup_used_gb", Labels: []string{"tenant", "tenant_id"}, Fn: nil, Slow: true},
}

func NewCinderExporter(config *ExporterConfig, logger log.Logger) (*CinderExporter, error) {
	exporter := CinderExporter{
		BaseOpenStackExporter{
			Name:           "cinder",
			ExporterConfig: *config,
			logger:         logger,
		},
	}

	for _, metric := range defaultCinderMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}

	return &exporter, nil
}

func ListVolumesStatus(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	type VolumeWithExt struct {
		volumes.Volume
		volumetenants.VolumeTenantExt
	}

	var allVolumes []VolumeWithExt
	var volumeListOption volumes.ListOpts

	if exporter.TenantID == "" {
		volumeListOption = volumes.ListOpts{AllTenants: true}
	} else {
		volumeListOption = volumes.ListOpts{TenantID: exporter.TenantID}
	}

	allPagesVolumes, err := volumes.List(exporter.Client, volumeListOption).AllPages()
	if err != nil {
		return err
	}

	err = volumes.ExtractVolumesInto(allPagesVolumes, &allVolumes)
	if err != nil {
		return err
	}

	// Volume status metrics
	for _, volume := range allVolumes {
		if len(volume.Attachments) > 0 {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["volume_status"].Metric,
				prometheus.GaugeValue, float64(mapVolumeStatus(volume.Status)), volume.ID, volume.Name,
				volume.Status, volume.Bootable, volume.TenantID, strconv.Itoa(volume.Size), volume.VolumeType, volume.Attachments[0].ServerID)
		} else {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["volume_status"].Metric,
				prometheus.GaugeValue, float64(mapVolumeStatus(volume.Status)), volume.ID, volume.Name,
				volume.Status, volume.Bootable, volume.TenantID, strconv.Itoa(volume.Size), volume.VolumeType, "")
		}
	}
	return nil
}

func ListVolumes(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	type VolumeWithExt struct {
		volumes.Volume
		volumetenants.VolumeTenantExt
	}

	var allVolumes []VolumeWithExt

	allPagesVolumes, err := volumes.List(exporter.Client, volumes.ListOpts{
		AllTenants: true,
	}).AllPages()
	if err != nil {
		return err
	}

	err = volumes.ExtractVolumesInto(allPagesVolumes, &allVolumes)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["volumes"].Metric,
		prometheus.GaugeValue, float64(len(allVolumes)))

	// Volume_gb metrics
	for _, volume := range allVolumes {
		if len(volume.Attachments) > 0 {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["volume_gb"].Metric,
				prometheus.GaugeValue, float64(volume.Size), volume.ID, volume.Name,
				volume.Status, volume.AvailabilityZone, volume.Bootable, volume.TenantID, volume.UserID, volume.VolumeType, volume.Attachments[0].ServerID)
		} else {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["volume_gb"].Metric,
				prometheus.GaugeValue, float64(volume.Size), volume.ID, volume.Name,
				volume.Status, volume.AvailabilityZone, volume.Bootable, volume.TenantID, volume.UserID, volume.VolumeType, "")
		}
	}

	volume_status_counter := map[string]int{
		"creating":          0,
		"available":         0,
		"reserved":          0,
		"attaching":         0,
		"detaching":         0,
		"in-use":            0,
		"maintenance":       0,
		"deleting":          0,
		"awaiting-transfer": 0,
		"error":             0,
		"error_deleting":    0,
		"backing-up":        0,
		"restoring-backup":  0,
		"error_backing-up":  0,
		"error_restoring":   0,
		"error_extending":   0,
		"downloading":       0,
		"uploading":         0,
		"retyping":          0,
		"extending":         0,
	}

	for _, volume := range allVolumes {
		volume_status_counter[volume.Status]++
	}

	// Volume status counter metrics
	for status, count := range volume_status_counter {
		ch <- prometheus.MustNewConstMetric(
			exporter.Metrics["volume_status_counter"].Metric,
			prometheus.GaugeValue,
			float64(count),
			status)
	}

	return nil
}

func ListSnapshots(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allSnapshots []snapshots.Snapshot

	allPagesSnapshot, err := snapshots.List(exporter.Client, snapshots.ListOpts{AllTenants: true}).AllPages()
	if err != nil {
		return err
	}

	allSnapshots, err = snapshots.ExtractSnapshots(allPagesSnapshot)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["snapshots"].Metric,
		prometheus.GaugeValue, float64(len(allSnapshots)))

	return nil
}

func ListCinderAgentState(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {

	var allServices []services.Service

	allPagesService, err := services.List(exporter.Client, services.ListOpts{}).AllPages()
	if err != nil {
		return err
	}
	allServices, err = services.ExtractServices(allPagesService)
	if err != nil {
		return err
	}

	for _, service := range allServices {
		var state = 0
		var id string

		if service.State == "up" {
			state = 1
		}
		if !exporter.DisableCinderAgentUUID {
			if id, err = exporter.UUIDGenFunc(); err != nil {
				return err
			}
		}

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["agent_state"].Metric,
			prometheus.CounterValue, float64(state), id, service.Host, service.Binary, service.Status, service.Zone, service.DisabledReason)
	}

	return nil
}

func ListCinderPoolCapacityFree(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	listOpts := schedulerstats.ListOpts{
		Detail: true,
	}

	allPages, err := schedulerstats.List(exporter.Client, listOpts).AllPages()
	if err != nil {
		return err
	}

	allStats, err := schedulerstats.ExtractStoragePools(allPages)
	if err != nil {
		return err
	}

	for _, stat := range allStats {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["pool_capacity_free_gb"].Metric, prometheus.GaugeValue,
			float64(stat.Capabilities.FreeCapacityGB), stat.Name, stat.Capabilities.VolumeBackendName, stat.Capabilities.VendorName)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["pool_capacity_total_gb"].Metric, prometheus.GaugeValue,
			float64(stat.Capabilities.TotalCapacityGB), stat.Name, stat.Capabilities.VolumeBackendName, stat.Capabilities.VendorName)
	}
	return nil
}

func ListVolumeLimits(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allProjects []projects.Project
	var eo gophercloud.EndpointOpts

	// We need a list of all tenants/projects. Therefore, within this nova exporter we need
	// to create an openstack client for the Identity/Keystone API.
	// If possible, use the EndpointOpts spefic to the identity service.
	if v, ok := endpointOpts["identity"]; ok {
		eo = v
	} else if v, ok := endpointOpts["volume"]; ok {
		eo = v
	} else {
		return errors.New("no EndpointOpts available to create Identity client")
	}

	c, err := openstack.NewIdentityV3(exporter.Client.ProviderClient, eo)
	if err != nil {
		return err
	}

	allPagesProject, err := projects.List(c, projects.ListOpts{DomainID: exporter.DomainID}).AllPages()
	if err != nil {
		return err
	}

	allProjects, err = projects.ExtractProjects(allPagesProject)
	if err != nil {
		return err
	}

	for _, p := range allProjects {
		// Limits are obtained from the cinder API, so now we can just use this exporter's client
		limits, err := quotasets.GetUsage(exporter.Client, p.ID).Extract()
		if err != nil {
			return err
		}

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["limits_volume_max_gb"].Metric,
			prometheus.GaugeValue, float64(limits.Gigabytes.Limit), p.Name, p.ID)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["limits_volume_used_gb"].Metric,
			prometheus.GaugeValue, float64(limits.Gigabytes.InUse), p.Name, p.ID)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["limits_backup_max_gb"].Metric,
			prometheus.GaugeValue, float64(limits.BackupGigabytes.Limit), p.Name, p.ID)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["limits_backup_used_gb"].Metric,
			prometheus.GaugeValue, float64(limits.BackupGigabytes.InUse), p.Name, p.ID)

	}

	return nil
}
