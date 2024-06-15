package openstack

import (
	"github.com/go-kit/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/db/v1/datastores"
	"github.com/gophercloud/gophercloud/openstack/db/v1/instances"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/prometheus/client_golang/prometheus"
)

var instance_status = []string{
	"NEW",              // The database instance creation request is just received by Trove.
	"BUILD",            // The database instance is being installed.
	"ACTIVE",           // The database instance is up and running.
	"REBOOT",           // The database instance is rebooting.
	"RESIZE",           // The database instance is being resized.
	"UPGRADE",          // The database instance is upgrading its datastore, e.g. from mysql 5.7.29 to mysql 5.7.30
	"RESTART_REQUIRED", // The database service needs to restart, e.g. due to the configuration change.
	"PROMOTE",          // A replica instance in the replication cluster is being promoted to the primary.
	"EJECT",            // The current primary instance in a replication cluster is being ejected, one of the replicas is going to be elected as the new primary.
	"DETACH",           // One of the replicas in a replication cluster is being detached and will become a standalone instance.
	"SHUTDOWN",         // The database instance is being shutdown during deletion.
	"BACKUP",           // The database instance is being backed up.
	"ERROR",            // The database instance is in error.
}

func mapInstanceStatus(current string) int {
	for idx, status := range instance_status {
		if current == status {
			return idx
		}
	}
	return -1
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
func extractInstances(r pagination.Page) ([]instanceAttributesExt, error) {
	var s struct {
		Instances []instanceAttributesExt `json:"instances"`
	}
	err := (r.(instances.InstancePage)).ExtractInto(&s)
	return s.Instances, err
}

// list retrieves the status and information for all database instances.
func list(client *gophercloud.ServiceClient) pagination.Pager {
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

func NewTroveExporter(config *ExporterConfig, logger log.Logger) (*TroveExporter, error) {
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

func ListAllInstances(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allInstances []instanceAttributesExt
	allPagesInstances, err := list(exporter.Client).AllPages()
	if err != nil {
		return err
	}
	allInstances, err = extractInstances(allPagesInstances)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["total_instances"].Metric,
		prometheus.GaugeValue, float64(len(allInstances)))
	for _, instance := range allInstances {
		labelValues := []string{instance.Datastore.Type, instance.Datastore.Version,
			instance.HealthStatus, instance.ID, instance.Name, instance.Region, instance.Status, instance.TenantID}
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["instance_status"].Metric,
			prometheus.GaugeValue, float64(mapInstanceStatus(instance.Status)), labelValues...)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["instance_volume_size_gb"].Metric,
			prometheus.GaugeValue, float64(instance.Volume.Size), labelValues...)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["instance_volume_used_gb"].Metric,
			prometheus.GaugeValue, instance.Volume.Used, labelValues...)
	}

	return nil
}
