package exporters

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/shares"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterTypedExporter("sharev2", NewManilaExporter)
}

type ManilaExporter struct {
	BaseOpenStackExporter
	sched Schedule
	descs manilaDescs
}

type manilaDescs struct {
	SharesCounter      *prometheus.Desc `metric:"shares_counter"`
	ShareGB            *prometheus.Desc `metric:"share_gb"            labels:"id,name,status,availability_zone,share_type,share_proto,share_type_name,project_id"`
	ShareStatus        *prometheus.Desc `metric:"share_status"        labels:"id,name,status,size,share_type,share_proto,share_type_name,project_id"`
	ShareStatusCounter *prometheus.Desc `metric:"share_status_counter" labels:"status"`
}

type manilaScrape struct {
	shares []shares.Share
}

var manilaGraph = Graph[*ManilaExporter, manilaScrape]{
	Sources: []Source[*ManilaExporter, manilaScrape]{
		{Name: "shares", Fetch: (*ManilaExporter).fetchShares},
	},
	Emitters: []Emitter[*ManilaExporter, manilaScrape]{
		{
			Name:    "share_counts",
			Metrics: []string{"shares_counter", "share_gb", "share_status_counter"},
			Sources: []string{"shares"},
			Emit:    (*ManilaExporter).emitShareCounts,
		},
		{
			Name:    "share_status",
			Metrics: []string{"share_status"},
			Sources: []string{"shares"},
			Emit:    (*ManilaExporter).emitShareStatus,
		},
	},
}

func NewManilaExporter(config *ExporterConfig, logger *slog.Logger) (*ManilaExporter, error) {
	e := &ManilaExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "sharev2",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	e.RegisterAndFillDescs(&e.descs)
	sched, err := manilaGraph.PruneSchedule(&e.BaseOpenStackExporter)
	if err != nil {
		return nil, err
	}
	e.sched = sched
	manilaGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)
	return e, nil
}

func (e *ManilaExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(manilaScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &manilaGraph, e.sched, s, ch)
	})
}

func (e *ManilaExporter) fetchShares(ctx context.Context, s *manilaScrape) error {
	allPages, err := shares.ListDetail(e.ClientV2, shares.ListOpts{AllTenants: true}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.shares, err = shares.ExtractShares(allPages)
	return err
}

var manilaShareStatuses = []string{
	"creating", "available", "updating", "migrating", "migration_error",
	"extending", "deleting", "shrinking", "error", "error_deleting",
	"shrinking_error", "reverting_error", "restoring", "reverting",
	"managing", "unmanaging", "reverting_to_snapshot", "soft_deleting", "inactive",
}

func (e *ManilaExporter) emitShareCounts(ctx context.Context, s *manilaScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.SharesCounter, float64(len(s.shares)))
	statusCounter := map[string]int{}
	for _, st := range manilaShareStatuses {
		statusCounter[st] = 0
	}
	for _, share := range s.shares {
		emitGauge(ch, e.descs.ShareGB, float64(share.Size), share.ID, share.Name, share.Status, share.AvailabilityZone,
			share.ShareType, share.ShareProto, share.ShareTypeName, share.ProjectID)
		statusCounter[share.Status]++
	}
	for status, count := range statusCounter {
		emitGauge(ch, e.descs.ShareStatusCounter, float64(count), status)
	}
	return nil
}

func (e *ManilaExporter) emitShareStatus(ctx context.Context, s *manilaScrape, ch chan<- prometheus.Metric) error {
	for _, share := range s.shares {
		emitGauge(ch, e.descs.ShareStatus,
			float64(mapVolumeStatus(share.Status)), share.ID, share.Name,
			share.Status, strconv.Itoa(share.Size), share.ShareType, share.ShareProto, share.ShareTypeName, share.ProjectID)
	}
	return nil
}
