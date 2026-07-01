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

func init() {
	RegisterTypedExporter("database", NewTroveExporter)
}

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

type instanceAttributesExt struct {
	ID           string
	Name         string
	Status       string
	Volume       instances.Volume
	Datastore    datastores.DatastorePartial
	Region       string
	HealthStatus string `json:"health_status"`
	TenantID     string `json:"tenant_id"`
}

func extractDBInstances(r pagination.Page) ([]instanceAttributesExt, error) {
	var s struct {
		Instances []instanceAttributesExt `json:"instances"`
	}
	err := (r.(instances.InstancePage)).ExtractInto(&s)
	return s.Instances, err
}

func listDBInstances(client *gophercloud.ServiceClient) pagination.Pager {
	return pagination.NewPager(client, client.ServiceURL("mgmt", "instances?include_clustered=False&deleted=False"), func(r pagination.PageResult) pagination.Page {
		return instances.InstancePage{LinkedPageBase: pagination.LinkedPageBase{PageResult: r}}
	})
}

type TroveExporter struct {
	BaseOpenStackExporter
	sched Schedule
	descs troveDescs
}

type troveDescs struct {
	TotalInstances       *prometheus.Desc `metric:"total_instances"`
	InstanceStatus       *prometheus.Desc `metric:"instance_status"       labels:"datastore_type,datastore_version,health_status,id,name,region,status,tenant_id"`
	InstanceVolumeSizeGB *prometheus.Desc `metric:"instance_volume_size_gb" labels:"datastore_type,datastore_version,health_status,id,name,region,status,tenant_id"`
	InstanceVolumeUsedGB *prometheus.Desc `metric:"instance_volume_used_gb" labels:"datastore_type,datastore_version,health_status,id,name,region,status,tenant_id"`
}

type troveScrape struct {
	instances []instanceAttributesExt
}

var troveGraph = Graph[*TroveExporter, troveScrape]{
	Sources: []Source[*TroveExporter, troveScrape]{
		{Name: "instances", Fetch: (*TroveExporter).fetchInstances},
	},
	Emitters: []Emitter[*TroveExporter, troveScrape]{
		{
			Name:    "instances",
			Metrics: []string{"total_instances", "instance_status", "instance_volume_size_gb", "instance_volume_used_gb"},
			Sources: []string{"instances"},
			Emit:    (*TroveExporter).emitInstances,
		},
	},
}

func NewTroveExporter(config *ExporterConfig, logger *slog.Logger) (*TroveExporter, error) {
	e := &TroveExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "trove",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	e.RegisterAndFillDescs(&e.descs)
	sched, err := troveGraph.PruneSchedule(&e.BaseOpenStackExporter)
	if err != nil {
		return nil, err
	}
	e.sched = sched
	troveGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)
	return e, nil
}

func (e *TroveExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(troveScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &troveGraph, e.sched, s, ch)
	})
}

func (e *TroveExporter) fetchInstances(ctx context.Context, s *troveScrape) error {
	allPages, err := listDBInstances(e.ClientV2).AllPages(ctx)
	if err != nil {
		return err
	}
	s.instances, err = extractDBInstances(allPages)
	return err
}

func (e *TroveExporter) emitInstances(ctx context.Context, s *troveScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.TotalInstances, float64(len(s.instances)))
	for _, inst := range s.instances {
		lbls := []string{inst.Datastore.Type, inst.Datastore.Version,
			inst.HealthStatus, inst.ID, inst.Name, inst.Region, inst.Status, inst.TenantID}
		emitGauge(ch, e.descs.InstanceStatus, float64(mapDBInstanceStatus(inst.Status)), lbls...)
		emitGauge(ch, e.descs.InstanceVolumeSizeGB, float64(inst.Volume.Size), lbls...)
		emitGauge(ch, e.descs.InstanceVolumeUsedGB, inst.Volume.Used, lbls...)
	}
	return nil
}
