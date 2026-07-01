package exporters

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterTypedExporter("image", NewGlanceExporter)
}

type GlanceExporter struct {
	BaseOpenStackExporter
	sched Schedule
	descs glanceDescs
}

type glanceDescs struct {
	Images         *prometheus.Desc `metric:"images"`
	ImageBytes     *prometheus.Desc `metric:"image_bytes"      labels:"id,name,tenant_id"                          slow:"true"`
	ImageCreatedAt *prometheus.Desc `metric:"image_created_at" labels:"id,name,tenant_id,visibility,hidden,status" slow:"true"`
}

type glanceScrape struct {
	images []images.Image
}

var glanceGraph = Graph[*GlanceExporter, glanceScrape]{
	Sources: []Source[*GlanceExporter, glanceScrape]{
		{Name: "images", Fetch: (*GlanceExporter).fetchImages},
	},
	Emitters: []Emitter[*GlanceExporter, glanceScrape]{
		{
			Name:    "count",
			Metrics: []string{"images"},
			Sources: []string{"images"},
			Emit:    (*GlanceExporter).emitCount,
		},
		{
			Name:    "properties",
			Metrics: []string{"image_bytes", "image_created_at"},
			Sources: []string{"images"},
			Emit:    (*GlanceExporter).emitProperties,
		},
	},
}

func NewGlanceExporter(config *ExporterConfig, logger *slog.Logger) (*GlanceExporter, error) {
	e := &GlanceExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "glance",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	e.RegisterAndFillDescs(&e.descs)
	sched, err := glanceGraph.PruneSchedule(&e.BaseOpenStackExporter)
	if err != nil {
		return nil, err
	}
	e.sched = sched
	glanceGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)
	return e, nil
}

func (e *GlanceExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(glanceScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &glanceGraph, e.sched, s, ch)
	})
}

func (e *GlanceExporter) fetchImages(ctx context.Context, s *glanceScrape) error {
	allPages, err := images.List(e.ClientV2, images.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.images, err = images.ExtractImages(allPages)
	return err
}

func (e *GlanceExporter) emitCount(ctx context.Context, s *glanceScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.Images, float64(len(s.images)))
	return nil
}

func (e *GlanceExporter) emitProperties(ctx context.Context, s *glanceScrape, ch chan<- prometheus.Metric) error {
	for _, img := range s.images {
		emitGauge(ch, e.descs.ImageBytes, float64(img.SizeBytes), img.ID, img.Name, img.Owner)
		emitGauge(ch, e.descs.ImageCreatedAt, float64(img.CreatedAt.Unix()), img.ID, img.Name,
			img.Owner, string(img.Visibility), strconv.FormatBool(img.Hidden), string(img.Status))
	}
	return nil
}
