package exporters

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/recordsets"
	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/zones"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

func init() {
	RegisterTypedExporter("dns", NewDesignateExporter)
}

var zoneStatuses = []string{"pending", "active", "deleted", "error"}
var recordsetStatuses = []string{"pending", "active", "deleted", "error"}

func mapZoneStatus(s string) int {
	for i, st := range zoneStatuses {
		if st == strings.ToLower(s) {
			return i
		}
	}
	return -1
}

func mapRecordsetStatus(s string) int {
	for i, st := range recordsetStatuses {
		if st == strings.ToLower(s) {
			return i
		}
	}
	return -1
}

type DesignateExporter struct {
	BaseOpenStackExporter
	sched Schedule
	descs designateDescs
}

type designateDescs struct {
	Zones            *prometheus.Desc `metric:"zones"`
	ZoneStatus       *prometheus.Desc `metric:"zone_status"       labels:"id,name,status,tenant_id,type"`
	Recordsets       *prometheus.Desc `metric:"recordsets"        labels:"zone_id,zone_name,tenant_id"`
	RecordsetsStatus *prometheus.Desc `metric:"recordsets_status" labels:"id,name,status,zone_id,zone_name,type"`
}

type designateScrape struct {
	zones      []zones.Zone
	recordsets []designateZoneRecordsets
}

type designateZoneRecordsets struct {
	zone       zones.Zone
	recordsets []recordsets.RecordSet
	ok         bool
}

var designateGraph = Graph[*DesignateExporter, designateScrape]{
	Sources: []Source[*DesignateExporter, designateScrape]{
		{Name: "zones", Fetch: (*DesignateExporter).fetchZones},
		{Name: "recordsets", DependsOn: []string{"zones"}, Fetch: (*DesignateExporter).fetchRecordsets},
	},
	Emitters: []Emitter[*DesignateExporter, designateScrape]{
		{
			Name:    "zones",
			Metrics: []string{"zones", "zone_status"},
			Sources: []string{"zones"},
			Emit:    (*DesignateExporter).emitZones,
		},
		{
			Name:    "recordsets",
			Metrics: []string{"recordsets", "recordsets_status"},
			Sources: []string{"recordsets"},
			Emit:    (*DesignateExporter).emitRecordsets,
		},
	},
}

func NewDesignateExporter(config *ExporterConfig, logger *slog.Logger) (*DesignateExporter, error) {
	e := &DesignateExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "designate",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	// Required header for all-projects zone collection.
	e.ClientV2.MoreHeaders = map[string]string{"X-Auth-All-Projects": "True"}

	e.RegisterAndFillDescs(&e.descs)
	sched, err := designateGraph.PruneSchedule(&e.BaseOpenStackExporter)
	if err != nil {
		return nil, err
	}
	e.sched = sched
	designateGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)
	return e, nil
}

func (e *DesignateExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(designateScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &designateGraph, e.sched, s, ch)
	})
}

func (e *DesignateExporter) fetchZones(ctx context.Context, s *designateScrape) error {
	allPages, err := zones.List(e.ClientV2, zones.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.zones, err = zones.ExtractZones(allPages)
	return err
}

func (e *DesignateExporter) fetchRecordsets(ctx context.Context, s *designateScrape) error {
	s.recordsets = make([]designateZoneRecordsets, len(s.zones))
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(e.GetDnsConcurrencyCount())

	for i, zone := range s.zones {
		i, zone := i, zone
		g.Go(func() error {
			allPages, err := recordsets.ListByZone(e.ClientV2, zone.ID, recordsets.ListOpts{}).AllPages(gCtx)
			if err != nil {
				return err
			}
			rsets, err := recordsets.ExtractRecordSets(allPages)
			if err != nil {
				return err
			}
			s.recordsets[i] = designateZoneRecordsets{zone: zone, recordsets: rsets, ok: true}
			return nil
		})
	}
	return g.Wait()
}

func (e *DesignateExporter) emitZones(ctx context.Context, s *designateScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.Zones, float64(len(s.zones)))
	for _, zone := range s.zones {
		emitGauge(ch, e.descs.ZoneStatus, float64(mapZoneStatus(zone.Status)), zone.ID, zone.Name, zone.Status, zone.ProjectID, zone.Type)
	}
	return nil
}

func (e *DesignateExporter) emitRecordsets(ctx context.Context, s *designateScrape, ch chan<- prometheus.Metric) error {
	for _, entry := range s.recordsets {
		if !entry.ok {
			continue
		}
		zone := entry.zone
		emitGauge(ch, e.descs.Recordsets, float64(len(entry.recordsets)), zone.ID, zone.Name, zone.ProjectID)
		for _, rs := range entry.recordsets {
			emitGauge(ch, e.descs.RecordsetsStatus,
				float64(mapRecordsetStatus(rs.Status)), rs.ID, rs.Name, rs.Status, rs.ZoneID, rs.ZoneName, rs.Type)
		}
	}
	return nil
}
