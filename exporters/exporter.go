package exporters

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"log/slog"

	gophercloudv2 "github.com/gophercloud/gophercloud/v2"
	clientutilsv2 "github.com/gophercloud/utils/v2/client"
	clientconfigv2 "github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/hashicorp/go-uuid"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

type Metric struct {
	Name              string
	Labels            []string
	Fn                ListFunc
	Slow              bool
	DeprecatedVersion string
}

const (
	//nolint: deadcode, unused
	BYTE = 1 << (10 * iota)
	//nolint: deadcode, unused
	KILOBYTE
	MEGABYTE
	GIGABYTE
	//nolint: deadcode, unused
	TERABYTE
)

var SupportedExporters = []string{"network", "compute", "image", "volume", "identity", "object-store", "load-balancer", "container-infra", "dns", "baremetal", "gnocchi", "database", "orchestration", "placement", "sharev2"}

type OpenStackExporter interface {
	prometheus.Collector

	GetName() string
	AddMetric(name string, fn ListFunc, labels []string, deprecatedVersion string, constLabels prometheus.Labels)
	MetricIsDisabled(name string) bool
}

func EnableExporter(service, prefix, cloud string, disabledMetrics []string, endpointType string, collectTime bool, disableSlowMetrics bool, disableDeprecatedMetrics bool, disableCinderAgentUUID bool, domainID string, tenantID string, novaMetadataMapping *utils.LabelMappingFlag, dnsConcurrentCount int, uuidGenFunc func() (string, error), logger *slog.Logger) (*OpenStackExporter, error) {
	exporter, err := NewExporter(service, prefix, cloud, disabledMetrics, endpointType, collectTime, disableSlowMetrics, disableDeprecatedMetrics, disableCinderAgentUUID, domainID, tenantID, novaMetadataMapping, dnsConcurrentCount, uuidGenFunc, logger)
	if err != nil {
		return nil, err
	}
	return &exporter, nil
}

type PrometheusMetric struct {
	Metric *prometheus.Desc
	Fn     ListFunc
}

type ExporterConfig struct {
	ClientV2                 *gophercloudv2.ServiceClient
	ServiceName              string
	Prefix                   string
	DisabledMetrics          []string
	CollectTime              bool
	UUIDGenFunc              func() (string, error)
	DisableSlowMetrics       bool
	DisableDeprecatedMetrics bool
	DisableCinderAgentUUID   bool
	DomainID                 string
	TenantID                 string
	NovaMetadataMapping      *utils.LabelMappingFlag
	DnsConcurrentCount       int
}

type BaseOpenStackExporter struct {
	ExporterConfig
	Name    string
	Metrics map[string]*PrometheusMetric
	logger  *slog.Logger
}

type ListFunc func(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error

var (
	endpointOptsV2   map[string]gophercloudv2.EndpointOpts
	endpointOptsV2Mu sync.Mutex
)

func (exporter *BaseOpenStackExporter) GetName() string {
	return fmt.Sprintf("%s_%s", exporter.Prefix, exporter.Name)
}

func (exporter *BaseOpenStackExporter) MetricIsDisabled(name string) bool {
	for _, metric := range exporter.DisabledMetrics {
		if metric == fmt.Sprintf("%s-%s", exporter.Name, name) {
			return true
		}
	}
	return false
}

func (exporter *BaseOpenStackExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range exporter.Metrics {
		ch <- metric.Metric
	}
}

func (exporter *BaseOpenStackExporter) RunCollection(metric *PrometheusMetric, metricName string, ch chan<- prometheus.Metric, logger *slog.Logger) error {
	ctx := context.TODO()

	exporter.logger.Info("Collecting metrics for exporter", "exporter", exporter.GetName(), "metrics", metricName)
	now := time.Now()
	err := metric.Fn(ctx, exporter, ch)
	if err != nil {
		return fmt.Errorf("failed to collect metric: %s, error: %s", metricName, err)
	}

	exporter.logger.Info("Collected metrics for exporter", "exporter", exporter.GetName(), "metrics", metricName)
	if exporter.CollectTime {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["openstack_metric_collect_seconds"].Metric, prometheus.GaugeValue, time.Since(now).Seconds(), metricName)
	}

	return nil
}

func (exporter *BaseOpenStackExporter) Collect(ch chan<- prometheus.Metric) {
	metricsCount := 0
	var failures int32

	var g errgroup.Group

	for name, metric := range exporter.Metrics {
		if metric.Fn == nil {
			exporter.logger.Debug("No function handler set for metric", "metric", name)
			continue
		}

		metricsCount++

		name := name
		metric := metric

		g.Go(func() error {
			if err := exporter.RunCollection(metric, name, ch, exporter.logger); err != nil {
				exporter.logger.Error(
					"Failed to collect metric for exporter",
					"exporter", exporter.Name,
					"metric", name,
					"err", err,
				)
				atomic.AddInt32(&failures, 1)
			}
			return nil
		})
	}

	_ = g.Wait()

	if metricsCount == 0 {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["up"].Metric, prometheus.GaugeValue, 0)
		return
	}

	if int(atomic.LoadInt32(&failures)) >= metricsCount {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["up"].Metric, prometheus.GaugeValue, 0)
	} else {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["up"].Metric, prometheus.GaugeValue, 1)
	}
}

func (exporter *BaseOpenStackExporter) isSlowMetric(metric *Metric) bool {
	return exporter.DisableSlowMetrics && metric.Slow
}

func (exporter *BaseOpenStackExporter) isDeprecatedMetric(metric *Metric) bool {
	return exporter.DisableDeprecatedMetrics && len(metric.DeprecatedVersion) > 0
}

func (exporter *BaseOpenStackExporter) AddMetric(name string, fn ListFunc, labels []string, deprecatedVersion string, constLabels prometheus.Labels) {
	if exporter.MetricIsDisabled(name) {
		exporter.logger.Warn("metric has been disabled for exporter, not collecting metrics", "metric", name, "exporter", exporter.Name)
		return
	}

	if len(deprecatedVersion) > 0 {
		exporter.logger.Warn("metric has been deprecated on exporter in version and it will be removed in next release", "metric", name, "exporter", exporter.Name, "version", deprecatedVersion)
	}

	if exporter.Metrics == nil {
		exporter.Metrics = make(map[string]*PrometheusMetric)
		exporter.Metrics["up"] = &PrometheusMetric{
			Metric: prometheus.NewDesc(
				prometheus.BuildFQName(exporter.GetName(), "", "up"),
				"up", nil, constLabels),
			Fn: nil,
		}
		exporter.Metrics["openstack_metric_collect_seconds"] = &PrometheusMetric{
			Metric: prometheus.NewDesc(
				"openstack_metric_collect_seconds", "Time needed to collect metric from OpenStack API", []string{"openstack_metric"}, prometheus.Labels{"openstack_service": exporter.GetName()}),
			Fn: nil,
		}
	}

	if constLabels == nil {
		constLabels = prometheus.Labels{}
	}

	// @TODO: get the region. constLabels["region"] = exporter.

	if _, ok := exporter.Metrics[name]; !ok {
		exporter.logger.Info("Adding metric to exporter", "metric", name, "exporter", exporter.Name)
		exporter.Metrics[name] = &PrometheusMetric{
			Metric: prometheus.NewDesc(
				prometheus.BuildFQName(exporter.GetName(), "", name),
				name, labels, constLabels),
			Fn: fn,
		}
	}
}

func (exporter *BaseOpenStackExporter) GetDnsConcurrencyCount() int {
	return exporter.DnsConcurrentCount
}

func NewExporter(name, prefix, cloud string, disabledMetrics []string, endpointType string, collectTime bool, disableSlowMetrics bool, disableDeprecatedMetrics bool, disableCinderAgentUUID bool, domainID string, tenantID string, novaMetadataMapping *utils.LabelMappingFlag, dnsConcurrentCount int, uuidGenFunc func() (string, error), logger *slog.Logger) (OpenStackExporter, error) {
	var exporter OpenStackExporter
	var err error
	var transport http.RoundTripper

	optsv2 := clientconfigv2.ClientOpts{Cloud: cloud}

	if _, ok := os.LookupEnv("OS_DEBUG"); ok {
		transport = &clientutilsv2.RoundTripper{
			Rt:     getNewHttpTransport(),
			Logger: &clientutilsv2.DefaultLogger{},
		}
	}

	clientV2, err := NewServiceClientV2(name, &optsv2, transport, endpointType)
	if err != nil {
		return nil, err
	}

	if uuidGenFunc == nil {
		uuidGenFunc = uuid.GenerateUUID
	}

	exporterConfig := ExporterConfig{
		ClientV2:                 clientV2,
		ServiceName:              name,
		Prefix:                   prefix,
		DisabledMetrics:          disabledMetrics,
		CollectTime:              collectTime,
		UUIDGenFunc:              uuidGenFunc,
		DisableSlowMetrics:       disableSlowMetrics,
		DisableDeprecatedMetrics: disableDeprecatedMetrics,
		DisableCinderAgentUUID:   disableCinderAgentUUID,
		DomainID:                 domainID,
		TenantID:                 tenantID,
		NovaMetadataMapping:      novaMetadataMapping,
		DnsConcurrentCount:       dnsConcurrentCount,
	}

	switch name {
	case "network":
		exporter, err = NewNeutronExporter(&exporterConfig, logger)
	case "compute":
		exporter, err = NewNovaExporter(&exporterConfig, logger)
	case "image":
		exporter, err = NewGlanceExporter(&exporterConfig, logger)
	case "volume":
		exporter, err = NewCinderExporter(&exporterConfig, logger)
	case "identity":
		exporter, err = NewKeystoneExporter(&exporterConfig, logger)
	case "object-store":
		exporter, err = NewObjectStoreExporter(&exporterConfig, logger)
	case "load-balancer":
		exporter, err = NewLoadbalancerExporter(&exporterConfig, logger)
	case "container-infra":
		exporter, err = NewContainerInfraExporter(&exporterConfig, logger)
	case "dns":
		exporter, err = NewDesignateExporter(&exporterConfig, logger)
	case "baremetal":
		exporter, err = NewIronicExporter(&exporterConfig, logger)
	case "gnocchi":
		exporter, err = NewGnocchiExporter(&exporterConfig, logger)
	case "database":
		exporter, err = NewTroveExporter(&exporterConfig, logger)
	case "orchestration":
		exporter, err = NewHeatExporter(&exporterConfig, logger)
	case "placement":
		exporter, err = NewPlacementExporter(&exporterConfig, logger)
	case "sharev2":
		exporter, err = NewManilaExporter(&exporterConfig, logger)
	default:
		return nil, fmt.Errorf("couldn't find a handler for %s exporter", name)
	}

	if err != nil {
		return nil, err
	}

	return exporter, nil
}
