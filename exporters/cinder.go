package exporters

import (
	"context"
	"strconv"
	"strings"

	"log/slog"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/quotasets"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/schedulerstats"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/services"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/prometheus/client_golang/prometheus"
)

type CinderExporter struct {
	BaseOpenStackExporter
}

var knownVolumeStatuses = map[string]int{
	"creating":          0,
	"available":         1,
	"reserved":          2,
	"attaching":         3,
	"detaching":         4,
	"in-use":            5,
	"maintenance":       6,
	"deleting":          7,
	"awaiting-transfer": 8,
	"error":             9,
	"error_deleting":    10,
	"backing-up":        11,
	"restoring-backup":  12,
	"error_backing-up":  13,
	"error_restoring":   14,
	"error_extending":   15,
	"downloading":       16,
	"uploading":         17,
	"retyping":          18,
	"extending":         19,
}

func mapVolumeStatus(volStatus string) int {
	v, ok := knownVolumeStatuses[strings.ToLower(volStatus)]
	if !ok {
		return -1
	}

	return v
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
	{Name: "volume_type_quota_gigabytes", Labels: []string{"tenant", "tenant_id", "volume_type"}, Fn: nil, Slow: true},
}

func NewCinderExporter(config *ExporterConfig, logger *slog.Logger) (*CinderExporter, error) {
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

func ListVolumesStatus(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	type VolumeWithExt = volumes.Volume

	var allVolumes []VolumeWithExt
	var volumeListOption volumes.ListOpts

	if exporter.TenantID == "" {
		volumeListOption = volumes.ListOpts{AllTenants: true}
	} else {
		volumeListOption = volumes.ListOpts{TenantID: exporter.TenantID}
	}

	allPagesVolumes, err := volumes.List(exporter.ClientV2, volumeListOption).AllPages(ctx)
	if err != nil {
		return err
	}

	err = volumes.ExtractVolumesInto(allPagesVolumes, &allVolumes)
	if err != nil {
		return err
	}

	// Volume status metrics
	for _, volume := range allVolumes {
		serverID := ""
		if len(volume.Attachments) > 0 {
			serverID = volume.Attachments[0].ServerID
		}

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["volume_status"].Metric,
			prometheus.GaugeValue, float64(mapVolumeStatus(volume.Status)), volume.ID, volume.Name,
			volume.Status, volume.Bootable, volume.TenantID, strconv.Itoa(volume.Size), volume.VolumeType, serverID)
	}

	return nil
}

func ListVolumes(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	type VolumeWithExt = volumes.Volume

	var allVolumes []VolumeWithExt

	allPagesVolumes, err := volumes.List(exporter.ClientV2, volumes.ListOpts{
		AllTenants: true,
	}).AllPages(ctx)
	if err != nil {
		return err
	}

	err = volumes.ExtractVolumesInto(allPagesVolumes, &allVolumes)
	if err != nil {
		return err
	}

	volume_status_counter := make(map[string]int, len(knownVolumeStatuses))
	for k := range knownVolumeStatuses {
		volume_status_counter[k] = 0
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["volumes"].Metric,
		prometheus.GaugeValue, float64(len(allVolumes)))

	for _, volume := range allVolumes {
		serverID := ""
		if len(volume.Attachments) > 0 {
			serverID = volume.Attachments[0].ServerID
		}

		// Volume_gb metrics
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["volume_gb"].Metric,
			prometheus.GaugeValue, float64(volume.Size), volume.ID, volume.Name,
			volume.Status, volume.AvailabilityZone, volume.Bootable, volume.TenantID, volume.UserID, volume.VolumeType, serverID)

		// collect statuses
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

func ListSnapshots(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allSnapshots []snapshots.Snapshot

	allPagesSnapshot, err := snapshots.List(exporter.ClientV2, snapshots.ListOpts{AllTenants: true}).AllPages(ctx)
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

func ListCinderAgentState(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allServices []services.Service

	allPagesService, err := services.List(exporter.ClientV2, services.ListOpts{}).AllPages(ctx)
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

func ListCinderPoolCapacityFree(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	listOpts := schedulerstats.ListOpts{
		Detail: true,
	}

	allPages, err := schedulerstats.List(exporter.ClientV2, listOpts).AllPages(ctx)
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

func ListVolumeLimits(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allProjects []projects.Project

	cli, err := newIdentityV3ClientV2FromExporter(exporter, "volume")
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
		// Limits are obtained from the cinder API, so now we can just use this exporter's client
		limits, err := quotasets.GetUsage(ctx, exporter.ClientV2, p.ID).Extract()
		if err != nil {
			return err
		}

		// Quotas are obtained from the cinder API
		quotas_p, err := quotasets.Get(exporter.Client, p.ID).Extract()
		if err != nil {
			return err
		}
		quotas := *quotas_p

		// Loop through all Extra quotas to automatically detect volume types
		for key, value := range quotas.Extra {
			if strings.HasPrefix(key, "gigabytes_") {
				volumeType := strings.TrimPrefix(key, "gigabytes_")
				if quotaValue, ok := value.(float64); ok {
					ch <- prometheus.MustNewConstMetric(exporter.Metrics["volume_type_quota_gigabytes"].Metric,
						prometheus.GaugeValue, quotaValue, p.Name, p.ID, volumeType)
				}
			}
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
