package exporters

import (
	"context"
	"log/slog"

	"github.com/gophercloud/utils/v2/gnocchi/metric/v1/metrics"
	"github.com/gophercloud/utils/v2/gnocchi/metric/v1/status"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterTypedExporter("gnocchi", NewGnocchiExporter)
}

type GnocchiExporter struct {
	BaseOpenStackExporter
	sched Schedule
	descs gnocchiDescs
}

type gnocchiDescs struct {
	StatusMetricdProcessors             *prometheus.Desc `metric:"status_metricd_processors"`
	StatusMetricHavingMeasuresToProcess *prometheus.Desc `metric:"status_metric_having_measures_to_process"`
	StatusMeasuresToProcess             *prometheus.Desc `metric:"status_measures_to_process"`
	TotalMetrics                        *prometheus.Desc `metric:"total_metrics"`
}

type gnocchiScrape struct {
	metricStatus *status.Status
	totalMetrics int
}

var gnocchiGraph = Graph[*GnocchiExporter, gnocchiScrape]{
	Sources: []Source[*GnocchiExporter, gnocchiScrape]{
		{Name: "status", Fetch: (*GnocchiExporter).fetchStatus},
		{Name: "metrics", Fetch: (*GnocchiExporter).fetchMetrics},
	},
	Emitters: []Emitter[*GnocchiExporter, gnocchiScrape]{
		{
			Name:    "status",
			Metrics: []string{"status_metricd_processors", "status_metric_having_measures_to_process", "status_measures_to_process"},
			Sources: []string{"status"},
			Emit:    (*GnocchiExporter).emitStatus,
		},
		{
			Name:    "metrics",
			Metrics: []string{"total_metrics"},
			Sources: []string{"metrics"},
			Emit:    (*GnocchiExporter).emitMetrics,
		},
	},
}

func NewGnocchiExporter(config *ExporterConfig, logger *slog.Logger) (*GnocchiExporter, error) {
	e := &GnocchiExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "gnocchi",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	e.RegisterAndFillDescs(&e.descs)
	sched, err := gnocchiGraph.PruneSchedule(&e.BaseOpenStackExporter)
	if err != nil {
		return nil, err
	}
	e.sched = sched
	gnocchiGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)
	return e, nil
}

func (e *GnocchiExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(gnocchiScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &gnocchiGraph, e.sched, s, ch)
	})
}

func (e *GnocchiExporter) fetchStatus(ctx context.Context, s *gnocchiScrape) error {
	if !e.IsMetricEnabled("status_metricd_processors", "status_metric_having_measures_to_process", "status_measures_to_process") {
		return nil
	}
	details := true
	st, err := status.Get(ctx, e.ClientV2, status.GetOpts{Details: &details}).Extract()
	if err != nil {
		return err
	}
	s.metricStatus = st
	return nil
}

func (e *GnocchiExporter) fetchMetrics(ctx context.Context, s *gnocchiScrape) error {
	if !e.IsMetricEnabled("total_metrics") {
		return nil
	}
	allPages, err := metrics.List(e.ClientV2, metrics.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	all, err := metrics.ExtractMetrics(allPages)
	if err != nil {
		return err
	}
	s.totalMetrics = len(all)
	return nil
}

func (e *GnocchiExporter) emitStatus(ctx context.Context, s *gnocchiScrape, ch chan<- prometheus.Metric) error {
	if s.metricStatus == nil {
		return nil
	}
	emitGauge(ch, e.descs.StatusMetricdProcessors, float64(len(s.metricStatus.Metricd.Processors)))
	emitGauge(ch, e.descs.StatusMetricHavingMeasuresToProcess, float64(s.metricStatus.Storage.Summary.Metrics))
	emitGauge(ch, e.descs.StatusMeasuresToProcess, float64(s.metricStatus.Storage.Summary.Measures))
	return nil
}

func (e *GnocchiExporter) emitMetrics(ctx context.Context, s *gnocchiScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.TotalMetrics, float64(s.totalMetrics))
	return nil
}
