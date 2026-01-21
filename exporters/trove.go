package exporters

import (
	"context"
	"log/slog"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/db/v1/datastores"
	"github.com/gophercloud/gophercloud/v2/openstack/db/v1/instances"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/prometheus/client_golang/prometheus"
)

var knownDBInstanceStatuses = map[string]int{
	"NEW":              0,  // The database instance creation request is just received by Trove.
	"BUILD":            1,  // The database instance is being installed.
	"ACTIVE":           2,  // The database instance is up and running.
	"REBOOT":           3,  // The database instance is rebooting.
	"RESIZE":           4,  // The database instance is being resized.
	"UPGRADE":          5,  // The database instance is upgrading its datastore, e.g. from mysql 5.7.29 to mysql 5.7.30
	"RESTART_REQUIRED": 6,  // The database service needs to restart, e.g. due to the configuration change.
	"PROMOTE":          7,  // A replica instance in the replication cluster is being promoted to the primary.
	"EJECT":            8,  // The current primary instance in a replication cluster is being ejected, one of the replicas is going to be elected as the new primary.
	"DETACH":           9,  // One of the replicas in a replication cluster is being detached and will become a standalone instance.
	"SHUTDOWN":         10, // The database instance is being shutdown during deletion.
	"BACKUP":           11, // The database instance is being backed up.
	"ERROR":            12, // The database instance is in error.
}

func mapDBInstanceStatus(current string) int {
	return mapStatus(knownDBInstanceStatuses, current)
}

type TroveExporter struct {
	BaseOpenStackExporter
}

type instanceAttributesExt struct {
	// Indicates the unique identifier for the instance resource.
	ID string

	// The human-readable name of the instance.
	Name string

	// The build status of the instance.
	Status string

	// Information about the attached volume of the instance.
	Volume instances.Volume

	// Indicates how the instance stores data.
	Datastore datastores.DatastorePartial

	// Region is the region the instance is located in.
	Region string

	HealthStatus string `json:"health_status"`

	// TenantID is the id of the project that owns the instance.
	TenantID string `json:"tenant_id"`
}

// extractInstances will convert a generic pagination struct into a more
// relevant slice of instanceAttributesExt structs.
func extractDBInstances(r pagination.Page) ([]instanceAttributesExt, error) {
	var s struct {
		Instances []instanceAttributesExt `json:"instances"`
	}
	err := (r.(instances.InstancePage)).ExtractInto(&s)
	return s.Instances, err
}

// list retrieves the status and information for all database instances.
func listDBInstances(client *gophercloud.ServiceClient) pagination.Pager {
	return pagination.NewPager(client, client.ServiceURL("mgmt", "instances?include_clustered=False&deleted=False"), func(r pagination.PageResult) pagination.Page {
		return instances.InstancePage{LinkedPageBase: pagination.LinkedPageBase{PageResult: r}}
	})
}

var defaultTroveMetrics = []Metric{
	{Name: "total_instances", Fn: ListAllInstances},
	{Name: "instance_status", Labels: []string{"datastore_type", "datastore_version", "health_status", "id", "name", "region", "status", "tenant_id"}, Fn: nil},
	{Name: "instance_volume_size_gb", Labels: []string{"datastore_type", "datastore_version", "health_status", "id", "name", "region", "status", "tenant_id"}, Fn: nil},
	{Name: "instance_volume_used_gb", Labels: []string{"datastore_type", "datastore_version", "health_status", "id", "name", "region", "status", "tenant_id"}, Fn: nil},
}

func NewTroveExporter(config *ExporterConfig, logger *slog.Logger) (*TroveExporter, error) {
	exporter := TroveExporter{
		BaseOpenStackExporter{
			Name:           "trove",
			ExporterConfig: *config,
			logger:         logger,
		},
	}

	for _, metric := range defaultTroveMetrics {
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}

	return &exporter, nil
}

func ListAllInstances(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allInstances []instanceAttributesExt
	allPagesInstances, err := listDBInstances(exporter.ClientV2).AllPages(ctx)
	if err != nil {
		return err
	}

	allInstances, err = extractDBInstances(allPagesInstances)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["total_instances"].Metric,
		prometheus.GaugeValue, float64(len(allInstances)))

	for _, instance := range allInstances {
		labelValues := []string{instance.Datastore.Type, instance.Datastore.Version,
			instance.HealthStatus, instance.ID, instance.Name, instance.Region, instance.Status, instance.TenantID}
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["instance_status"].Metric,
			prometheus.GaugeValue, float64(mapDBInstanceStatus(instance.Status)), labelValues...)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["instance_volume_size_gb"].Metric,
			prometheus.GaugeValue, float64(instance.Volume.Size), labelValues...)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["instance_volume_used_gb"].Metric,
			prometheus.GaugeValue, instance.Volume.Used, labelValues...)
	}

	return nil
}
