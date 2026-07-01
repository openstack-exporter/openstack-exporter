package exporters

import (
	"context"
	"log/slog"

	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterTypedExporter("object-store", NewObjectStoreExporter)
}

type ObjectStoreExporter struct {
	BaseOpenStackExporter
	sched Schedule
	descs objectStoreDescs
}

type objectStoreDescs struct {
	Objects *prometheus.Desc `metric:"objects" labels:"container_name"`
	Bytes   *prometheus.Desc `metric:"bytes"   labels:"container_name"`
}

type objectStoreScrape struct {
	containers []containers.Container
}

var objectStoreGraph = Graph[*ObjectStoreExporter, objectStoreScrape]{
	Sources: []Source[*ObjectStoreExporter, objectStoreScrape]{
		{Name: "containers", Fetch: (*ObjectStoreExporter).fetchContainers},
	},
	Emitters: []Emitter[*ObjectStoreExporter, objectStoreScrape]{
		{
			Name:    "containers",
			Metrics: []string{"objects", "bytes"},
			Sources: []string{"containers"},
			Emit:    (*ObjectStoreExporter).emitContainers,
		},
	},
}

func NewObjectStoreExporter(config *ExporterConfig, logger *slog.Logger) (*ObjectStoreExporter, error) {
	e := &ObjectStoreExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "object_store",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	e.RegisterAndFillDescs(&e.descs)
	sched, err := objectStoreGraph.PruneSchedule(&e.BaseOpenStackExporter)
	if err != nil {
		return nil, err
	}
	e.sched = sched
	objectStoreGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)
	return e, nil
}

func (e *ObjectStoreExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(objectStoreScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &objectStoreGraph, e.sched, s, ch)
	})
}

func (e *ObjectStoreExporter) fetchContainers(ctx context.Context, s *objectStoreScrape) error {
	return containers.List(e.ClientV2, containers.ListOpts{}).EachPage(ctx,
		func(_ context.Context, page pagination.Page) (bool, error) {
			list, err := containers.ExtractInfo(page)
			if err != nil {
				return false, err
			}
			s.containers = append(s.containers, list...)
			return true, nil
		})
}

func (e *ObjectStoreExporter) emitContainers(ctx context.Context, s *objectStoreScrape, ch chan<- prometheus.Metric) error {
	for _, c := range s.containers {
		emitGauge(ch, e.descs.Objects, float64(c.Count), c.Name)
		emitGauge(ch, e.descs.Bytes, float64(c.Bytes), c.Name)
	}
	return nil
}
