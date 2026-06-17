package exporters

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/quotasets"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/schedulerstats"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/services"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterTypedExporter("volume", NewCinderExporter)
}

// CinderExporter exports Cinder (Block Storage) metrics.
type CinderExporter struct {
	BaseOpenStackExporter
	sched Schedule
	descs cinderDescs
}

// cinderDescs holds pre-resolved prometheus descriptors for Cinder metrics.
// Fields are populated by RegisterAndFillDescs using struct tags at construction
// time, so emitters can emit metrics without a map lookup on the hot path.
// A nil field means the metric was disabled/slow/deprecated and should be skipped.
type cinderDescs struct {
	Volumes             *prometheus.Desc `metric:"volumes"`
	VolumeGB            *prometheus.Desc `metric:"volume_gb"            labels:"id,name,status,availability_zone,bootable,tenant_id,user_id,volume_type,server_id"`
	VolumeStatus        *prometheus.Desc `metric:"volume_status"        labels:"id,name,status,bootable,tenant_id,size,volume_type,server_id"                         deprecated:"1.4"`
	VolumeStatusCounter *prometheus.Desc `metric:"volume_status_counter" labels:"status"`
	Snapshots           *prometheus.Desc `metric:"snapshots"`
	PoolCapacityFreeGB  *prometheus.Desc `metric:"pool_capacity_free_gb"  labels:"name,volume_backend_name,vendor_name"`
	PoolCapacityTotalGB *prometheus.Desc `metric:"pool_capacity_total_gb" labels:"name,volume_backend_name,vendor_name"`
	AgentState          *prometheus.Desc `metric:"agent_state"           labels:"uuid,hostname,service,adminState,zone,disabledReason"`
	LimitsVolumeMaxGB   *prometheus.Desc `metric:"limits_volume_max_gb"  labels:"tenant,tenant_id" slow:"true"`
	LimitsVolumeUsedGB  *prometheus.Desc `metric:"limits_volume_used_gb" labels:"tenant,tenant_id" slow:"true"`
	LimitsBackupMaxGB   *prometheus.Desc `metric:"limits_backup_max_gb"  labels:"tenant,tenant_id" slow:"true"`
	LimitsBackupUsedGB  *prometheus.Desc `metric:"limits_backup_used_gb" labels:"tenant,tenant_id" slow:"true"`
	VolumeTypeQuota     *prometheus.Desc `metric:"volume_type_quota_gigabytes" labels:"tenant,tenant_id,volume_type" slow:"true"`
}

// cinderScrape holds typed data fetched during a single Collect call.
// Each field is written by exactly one Source; emitters read fields after their
// declared source dependencies complete.
type cinderScrape struct {
	volumes   []volumes.Volume
	snapshots []snapshots.Snapshot
	pools     []schedulerstats.StoragePool
	agents    []services.Service
	limits    []cinderProjectLimits
}

// cinderProjectLimits bundles per-project quota data fetched in fetchLimits.
type cinderProjectLimits struct {
	projectName string
	projectID   string
	usage       quotasets.QuotaUsageSet
	quotas      quotasets.QuotaSet
}

// cinderGraph declares the DAG topology: which sources feed which emitters.
var cinderGraph = Graph[*CinderExporter, cinderScrape]{
	Sources: []Source[*CinderExporter, cinderScrape]{
		{Name: "volumes", Fetch: (*CinderExporter).fetchVolumes},
		{Name: "snapshots", Fetch: (*CinderExporter).fetchSnapshots},
		{Name: "pools", Fetch: (*CinderExporter).fetchPools},
		{Name: "agents", Fetch: (*CinderExporter).fetchAgents},
		{Name: "limits", Fetch: (*CinderExporter).fetchLimits},
	},
	Emitters: []Emitter[*CinderExporter, cinderScrape]{
		{
			Name:    "volumes",
			Metrics: []string{"volumes", "volume_gb", "volume_status_counter"},
			Sources: []string{"volumes"},
			Emit:    (*CinderExporter).emitVolumes,
		},
		{
			Name:    "volumeStatus",
			Metrics: []string{"volume_status"},
			Sources: []string{"volumes"},
			Emit:    (*CinderExporter).emitVolumeStatus,
		},
		{
			Name:    "snapshots",
			Metrics: []string{"snapshots"},
			Sources: []string{"snapshots"},
			Emit:    (*CinderExporter).emitSnapshots,
		},
		{
			Name:    "pools",
			Metrics: []string{"pool_capacity_free_gb", "pool_capacity_total_gb"},
			Sources: []string{"pools"},
			Emit:    (*CinderExporter).emitPools,
		},
		{
			Name:    "agents",
			Metrics: []string{"agent_state"},
			Sources: []string{"agents"},
			Emit:    (*CinderExporter).emitAgents,
		},
		{
			Name:    "limits",
			Metrics: []string{"limits_volume_max_gb", "limits_volume_used_gb", "limits_backup_max_gb", "limits_backup_used_gb", "volume_type_quota_gigabytes"},
			Sources: []string{"limits"},
			Emit:    (*CinderExporter).emitLimits,
		},
	},
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
	return mapStatus(knownVolumeStatuses, volStatus)
}

// NewCinderExporter constructs a CinderExporter, registers metrics from
// cinderDescs struct tags, and computes a pruned DAG schedule.
func NewCinderExporter(config *ExporterConfig, logger *slog.Logger) (*CinderExporter, error) {
	e := &CinderExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "cinder",
			ExporterConfig: *config,
			logger:         logger,
		},
	}

	// RegisterAndFillDescs reads struct tags from cinderDescs, registers metrics
	// via AddMetric (respecting disabled/slow/deprecated flags), and sets each
	// *prometheus.Desc field — one declaration, no duplication.
	e.RegisterAndFillDescs(&e.descs)

	// Compute the pruned schedule from enabled metrics and DAG dependencies.
	sched, err := cinderGraph.PruneSchedule(&e.BaseOpenStackExporter)
	if err != nil {
		return nil, err
	}
	e.sched = sched
	cinderGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)

	return e, nil
}

// Collect implements prometheus.Collector via the DAG schedule.
func (e *CinderExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(cinderScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &cinderGraph, e.sched, s, ch)
	})
}

// ---------------------------------------------------------------------------
// Source methods — fetch data into cinderScrape.
// ---------------------------------------------------------------------------

func (e *CinderExporter) fetchVolumes(ctx context.Context, s *cinderScrape) error {
	allPages, err := volumes.List(e.ClientV2, getVolumeListOptions(e.TenantID)).AllPages(ctx)
	if err != nil {
		return err
	}
	return volumes.ExtractVolumesInto(allPages, &s.volumes)
}

func (e *CinderExporter) fetchSnapshots(ctx context.Context, s *cinderScrape) error {
	allPages, err := snapshots.List(e.ClientV2, getSnapshotListOptions(e.TenantID)).AllPages(ctx)
	if err != nil {
		return err
	}
	var err2 error
	s.snapshots, err2 = snapshots.ExtractSnapshots(allPages)
	return err2
}

func (e *CinderExporter) fetchPools(ctx context.Context, s *cinderScrape) error {
	allPages, err := schedulerstats.List(e.ClientV2, schedulerstats.ListOpts{Detail: true}).AllPages(ctx)
	if err != nil {
		return err
	}
	var err2 error
	s.pools, err2 = schedulerstats.ExtractStoragePools(allPages)
	return err2
}

func (e *CinderExporter) fetchAgents(ctx context.Context, s *cinderScrape) error {
	if !e.IsMetricEnabled("agent_state") {
		return nil
	}
	allPages, err := services.List(e.ClientV2, services.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	var err2 error
	s.agents, err2 = services.ExtractServices(allPages)
	return err2
}

func (e *CinderExporter) fetchLimits(ctx context.Context, s *cinderScrape) error {
	if !e.IsMetricEnabled("limits_volume_max_gb", "limits_volume_used_gb",
		"limits_backup_max_gb", "limits_backup_used_gb", "volume_type_quota_gigabytes") {
		return nil
	}
	allProjects, err := GetProjects(ctx, &e.BaseOpenStackExporter)
	if err != nil {
		return err
	}
	for _, p := range allProjects {
		usage, err := quotasets.GetUsage(ctx, e.ClientV2, p.ID).Extract()
		if err != nil {
			return err
		}
		quotasPtr, err := quotasets.Get(ctx, e.ClientV2, p.ID).Extract()
		if err != nil {
			return err
		}
		s.limits = append(s.limits, cinderProjectLimits{
			projectName: p.Name,
			projectID:   p.ID,
			usage:       usage,
			quotas:      *quotasPtr,
		})
	}
	return nil
}

// ---------------------------------------------------------------------------
// Emitter methods — read cinderScrape and emit to the prometheus channel.
// Each uses e.descs.* directly (no map lookup); nil check skips disabled metrics.
// ---------------------------------------------------------------------------

func (e *CinderExporter) emitVolumes(ctx context.Context, s *cinderScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.Volumes, float64(len(s.volumes)))

	statusCounter := make(map[string]int, len(knownVolumeStatuses))
	for k := range knownVolumeStatuses {
		statusCounter[k] = 0
	}

	for _, vol := range s.volumes {
		serverID := ""
		if len(vol.Attachments) > 0 {
			serverID = vol.Attachments[0].ServerID
		}
		emitGauge(ch, e.descs.VolumeGB, float64(vol.Size), vol.ID, vol.Name, vol.Status, vol.AvailabilityZone,
			vol.Bootable, vol.TenantID, vol.UserID, vol.VolumeType, serverID)
		statusCounter[vol.Status]++
	}

	for status, count := range statusCounter {
		emitGauge(ch, e.descs.VolumeStatusCounter, float64(count), status)
	}

	return nil
}

func (e *CinderExporter) emitVolumeStatus(ctx context.Context, s *cinderScrape, ch chan<- prometheus.Metric) error {
	for _, vol := range s.volumes {
		serverID := ""
		if len(vol.Attachments) > 0 {
			serverID = vol.Attachments[0].ServerID
		}
		emitGauge(ch, e.descs.VolumeStatus,
			float64(mapVolumeStatus(vol.Status)), vol.ID, vol.Name, vol.Status,
			vol.Bootable, vol.TenantID, strconv.Itoa(vol.Size), vol.VolumeType, serverID)
	}
	return nil
}

func (e *CinderExporter) emitSnapshots(ctx context.Context, s *cinderScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.Snapshots, float64(len(s.snapshots)))
	return nil
}

func (e *CinderExporter) emitPools(ctx context.Context, s *cinderScrape, ch chan<- prometheus.Metric) error {
	for _, pool := range s.pools {
		emitGauge(ch, e.descs.PoolCapacityFreeGB, float64(pool.Capabilities.FreeCapacityGB), pool.Name, pool.Capabilities.VolumeBackendName, pool.Capabilities.VendorName)
		emitGauge(ch, e.descs.PoolCapacityTotalGB, float64(pool.Capabilities.TotalCapacityGB), pool.Name, pool.Capabilities.VolumeBackendName, pool.Capabilities.VendorName)
	}
	return nil
}

func (e *CinderExporter) emitAgents(ctx context.Context, s *cinderScrape, ch chan<- prometheus.Metric) error {
	for _, agent := range s.agents {
		state := 0.0
		if agent.State == "up" {
			state = 1.0
		}
		id := ""
		if !e.DisableCinderAgentUUID {
			var err error
			if id, err = e.UUIDGenFunc(); err != nil {
				return err
			}
		}
		emitGauge(ch, e.descs.AgentState,
			state, id, agent.Host, agent.Binary, agent.Status, agent.Zone, agent.DisabledReason)
	}
	return nil
}

func (e *CinderExporter) emitLimits(ctx context.Context, s *cinderScrape, ch chan<- prometheus.Metric) error {
	for _, lim := range s.limits {
		emitGauge(ch, e.descs.LimitsVolumeMaxGB, float64(lim.usage.Gigabytes.Limit), lim.projectName, lim.projectID)
		emitGauge(ch, e.descs.LimitsVolumeUsedGB, float64(lim.usage.Gigabytes.InUse), lim.projectName, lim.projectID)
		emitGauge(ch, e.descs.LimitsBackupMaxGB, float64(lim.usage.BackupGigabytes.Limit), lim.projectName, lim.projectID)
		emitGauge(ch, e.descs.LimitsBackupUsedGB, float64(lim.usage.BackupGigabytes.InUse), lim.projectName, lim.projectID)
		for key, value := range lim.quotas.Extra {
			if strings.HasPrefix(key, "gigabytes_") {
				volumeType := strings.TrimPrefix(key, "gigabytes_")
				if quotaValue, ok := value.(float64); ok {
					emitGauge(ch, e.descs.VolumeTypeQuota, quotaValue, lim.projectName, lim.projectID, volumeType)
				}
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func getVolumeListOptions(tenantID string) volumes.ListOpts {
	if tenantID == "" {
		return volumes.ListOpts{AllTenants: true}
	}
	return volumes.ListOpts{TenantID: tenantID}
}

func getSnapshotListOptions(tenantID string) snapshots.ListOpts {
	if tenantID == "" {
		return snapshots.ListOpts{AllTenants: true}
	}
	return snapshots.ListOpts{TenantID: tenantID}
}
