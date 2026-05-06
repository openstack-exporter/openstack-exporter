package exporters

import (
	"errors"
	"strconv"
	"strings"

	"log/slog"

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

const CINDER_SERVICE string = "cinder"

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
			Name:           CINDER_SERVICE,
			ExporterConfig: *config,
			logger:         logger,
		},
	}

	for _, metric := range defaultCinderMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			labels := computeMetricLabels(CINDER_SERVICE, metric, exporter.ExtraLabels)
			constLabels := computeConstantLabels(CINDER_SERVICE, metric, exporter.ExtraLabels)
			exporter.AddMetric(metric.Name, metric.Fn, labels, metric.DeprecatedVersion, constLabels)
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
	extraLabels := make([]string, 0)
	labelSpec := exporter.ExtraLabels.Extract(CINDER_SERVICE, "volume_status")
	if labelSpec != nil {
		extraLabels = append(extraLabels, labelSpec.DynamicFields...)
	}

	// Volume status metrics
	for _, volume := range allVolumes {
		extraLabelValues := make([]string, len(extraLabels))
		for i, label := range extraLabels {
			extraLabelValues[i] = resolveField(volume, label)
		}
		labelValues := []string{volume.ID, volume.Name, volume.Status, volume.Bootable, volume.TenantID, strconv.Itoa(volume.Size), volume.VolumeType}
		if len(volume.Attachments) > 0 {

			labelValues = append(labelValues, volume.Attachments[0].ServerID)

		} else {
			labelValues = append(labelValues, "")

		}
		labelValues = append(labelValues, extraLabelValues...)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["volume_status"].Metric,
			prometheus.GaugeValue, float64(mapVolumeStatus(volume.Status)), labelValues...)
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

	extraLabels := make([]string, 0)
	labelSpec := exporter.ExtraLabels.Extract(CINDER_SERVICE, "volume_gb")
	if labelSpec != nil {
		extraLabels = append(extraLabels, labelSpec.DynamicFields...)
	}

	// Volume_gb metrics
	for _, volume := range allVolumes {
		extraLabelValues := make([]string, len(extraLabels))
		for i, label := range extraLabels {
			extraLabelValues[i] = resolveField(volume, label)
		}

		labelValues := []string{volume.ID, volume.Name, volume.Status, volume.AvailabilityZone, volume.Bootable, volume.TenantID, volume.UserID, volume.VolumeType}
		if len(volume.Attachments) > 0 {
			labelValues = append(labelValues, volume.Attachments[0].ServerID)
		} else {
			labelValues = append(labelValues, "")
		}
		labelValues = append(labelValues, extraLabelValues...)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["volume_gb"].Metric,
			prometheus.GaugeValue, float64(volume.Size), labelValues...)
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
	extraLabels := make([]string, 0)
	labelSpec := exporter.ExtraLabels.Extract(CINDER_SERVICE, "agent_state")
	if labelSpec != nil {
		extraLabels = append(extraLabels, labelSpec.DynamicFields...)
	}

	for _, service := range allServices {
		var state = 0
		var id string
		extraLabelValues := make([]string, len(extraLabels))
		for i, label := range extraLabels {
			extraLabelValues[i] = resolveField(service, label)
		}

		if service.State == "up" {
			state = 1
		}
		if !exporter.DisableCinderAgentUUID {
			if id, err = exporter.UUIDGenFunc(); err != nil {
				return err
			}
		}

		labelValues := []string{id, service.Host, service.Binary, service.Status, service.Zone, service.DisabledReason}
		labelValues = append(labelValues, extraLabelValues...)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["agent_state"].Metric,
			prometheus.CounterValue, float64(state), labelValues...)
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
	freeGbExtraLabels := make([]string, 0)
	freeGbLabelSpec := exporter.ExtraLabels.Extract(CINDER_SERVICE, "pool_capacity_free_gb")
	if freeGbLabelSpec != nil {
		freeGbExtraLabels = append(freeGbExtraLabels, freeGbLabelSpec.DynamicFields...)
	}
	totalGbExtraLabels := make([]string, 0)
	totalGbLabelSpec := exporter.ExtraLabels.Extract(CINDER_SERVICE, "pool_capacity_total_gb")
	if totalGbLabelSpec != nil {
		totalGbExtraLabels = append(totalGbExtraLabels, totalGbLabelSpec.DynamicFields...)
	}

	for _, stat := range allStats {
		freeGbExtraLabelValues := make([]string, len(freeGbExtraLabels))
		for i, label := range freeGbExtraLabels {
			freeGbExtraLabelValues[i] = resolveField(stat, label)
		}
		totalGbExtraLabelValues := make([]string, len(totalGbExtraLabels))
		for i, label := range totalGbExtraLabels {
			totalGbExtraLabelValues[i] = resolveField(stat, label)
		}
		labelValues := []string{stat.Name, stat.Capabilities.VolumeBackendName, stat.Capabilities.VendorName}
		freeGbLabelValues := append(labelValues, freeGbExtraLabelValues...)
		totalGbLabelValues := append(labelValues, totalGbExtraLabelValues...)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["pool_capacity_free_gb"].Metric, prometheus.GaugeValue,
			float64(stat.Capabilities.FreeCapacityGB), freeGbLabelValues...)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["pool_capacity_total_gb"].Metric, prometheus.GaugeValue,
			float64(stat.Capabilities.TotalCapacityGB), totalGbLabelValues...)
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

	limitsVolumeMaxSpec := exporter.ExtraLabels.Extract(CINDER_SERVICE, "limits_volume_max_gb")
	limitsVolumeUsedSpec := exporter.ExtraLabels.Extract(CINDER_SERVICE, "limits_volume_used_gb")
	limitsBackupMaxSpec := exporter.ExtraLabels.Extract(CINDER_SERVICE, "limits_backup_max_gb")
	limitsBackupUsedSpec := exporter.ExtraLabels.Extract(CINDER_SERVICE, "limits_backup_used_gb")
	volumeTypeQuotaSpec := exporter.ExtraLabels.Extract(CINDER_SERVICE, "volume_type_quota_gigabytes")
	for _, p := range allProjects {
		// Limits are obtained from the cinder API, so now we can just use this exporter's client
		limits, err := quotasets.GetUsage(exporter.Client, p.ID).Extract()
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
						prometheus.GaugeValue, quotaValue, append([]string{p.Name, p.ID, volumeType}, resolveExtraLabelValues(p, volumeTypeQuotaSpec)...)...)
				}
			}
		}

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["limits_volume_max_gb"].Metric,
			prometheus.GaugeValue, float64(limits.Gigabytes.Limit), append([]string{p.Name, p.ID}, resolveExtraLabelValues(p, limitsVolumeMaxSpec)...)...)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["limits_volume_used_gb"].Metric,
			prometheus.GaugeValue, float64(limits.Gigabytes.InUse), append([]string{p.Name, p.ID}, resolveExtraLabelValues(p, limitsVolumeUsedSpec)...)...)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["limits_backup_max_gb"].Metric,
			prometheus.GaugeValue, float64(limits.BackupGigabytes.Limit), append([]string{p.Name, p.ID}, resolveExtraLabelValues(p, limitsBackupMaxSpec)...)...)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["limits_backup_used_gb"].Metric,
			prometheus.GaugeValue, float64(limits.BackupGigabytes.InUse), append([]string{p.Name, p.ID}, resolveExtraLabelValues(p, limitsBackupUsedSpec)...)...)
	}

	return nil
}
