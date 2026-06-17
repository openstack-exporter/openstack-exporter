package exporters

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"log/slog"
	"slices"

	gophercloudv2 "github.com/gophercloud/gophercloud/v2"
	clientutilsv2 "github.com/gophercloud/utils/v2/client"
	clientconfigv2 "github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/hashicorp/go-uuid"
	"github.com/mitchellh/go-homedir"
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
	IsMetricEnabled(name string) bool
}

func EnableExporter(service string, opts ExporterOptions, logger *slog.Logger) (*OpenStackExporter, error) {
	exporter, err := NewExporter(service, opts, logger)
	if err != nil {
		return nil, err
	}
	return &exporter, nil
}

type PrometheusMetric struct {
	Metric *prometheus.Desc
	Fn     ListFunc
}

// ExporterOptions holds all user-supplied configuration for an exporter.
// It is passed directly to EnableExporter / NewExporter and is embedded in
// ExporterConfig, which adds the resolved service client and service name.
type ExporterOptions struct {
	// Cloud is the name of the cloud entry in clouds.yaml to scrape.
	Cloud string
	// Prefix is the metric name prefix (default: "openstack").
	Prefix string
	// DisabledMetrics is a list of "exporter-metric" keys to suppress entirely
	// (e.g. "cinder-snapshots"). Takes precedence over EnabledMetrics.
	DisabledMetrics []string
	// EnabledMetrics overrides DisableSlowMetrics / DisableDeprecatedMetrics for
	// individual metrics, using the same "exporter-metric" format
	// (e.g. "nova-limits_vcpus_max").
	EnabledMetrics []string
	// EndpointType selects the OpenStack endpoint to connect to
	// ("public", "internal", or "admin").
	EndpointType string
	// CollectTime enables emission of per-metric collection duration.
	CollectTime bool
	// DisableSlowMetrics suppresses metrics marked Slow: true in their
	// definition. Individual metrics can be re-enabled via EnabledMetrics.
	DisableSlowMetrics bool
	// DisableDeprecatedMetrics suppresses metrics that carry a DeprecatedVersion.
	// Individual metrics can be re-enabled via EnabledMetrics.
	DisableDeprecatedMetrics bool
	// DisableCinderAgentUUID disables UUID generation for Cinder agent metrics.
	DisableCinderAgentUUID bool
	// DomainID restricts metric collection to a single Keystone domain.
	// Empty string means all domains.
	DomainID string
	// TenantID restricts metric collection to a single project.
	// Empty string means all projects.
	TenantID string
	// NovaMetadataMapping maps Nova server metadata keys to extra Prometheus labels
	// on the openstack_nova_server_status metric.
	NovaMetadataMapping *utils.LabelMappingFlag
	// DnsConcurrentCount controls the number of concurrent requests used when
	// collecting DNS recordsets.
	DnsConcurrentCount int
	// UUIDGenFunc is the function used to generate UUIDs for Cinder agents.
	// Defaults to uuid.GenerateUUID when nil.
	UUIDGenFunc func() (string, error)
}

type ExporterConfig struct {
	ExporterOptions
	ClientV2    *gophercloudv2.ServiceClient
	ServiceName string
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

// qualifiedMetricName returns the backward-compatible "exporter-metric" key
// used in DisabledMetrics / EnabledMetrics lists (e.g. "nova-limits_vcpus_max").
func (exporter *BaseOpenStackExporter) qualifiedMetricName(name string) string {
	return exporter.Name + "-" + name
}

// isExplicitlyEnabled reports whether name appears in the EnabledMetrics list.
// An explicitly-enabled metric overrides global DisableSlowMetrics /
// DisableDeprecatedMetrics flags.
func (exporter *BaseOpenStackExporter) isExplicitlyEnabled(name string) bool {
	return slices.Contains(exporter.EnabledMetrics, exporter.qualifiedMetricName(name))
}

// IsMetricEnabled reports whether name should be collected and emitted.
// A metric is disabled when it appears in DisabledMetrics; an explicit entry
// in EnabledMetrics does NOT override a DisabledMetrics entry.
func (exporter *BaseOpenStackExporter) IsMetricEnabled(name string) bool {
	return !slices.Contains(exporter.DisabledMetrics, exporter.qualifiedMetricName(name))
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
	return exporter.DisableSlowMetrics && metric.Slow && !exporter.isExplicitlyEnabled(metric.Name)
}

func (exporter *BaseOpenStackExporter) isDeprecatedMetric(metric *Metric) bool {
	return exporter.DisableDeprecatedMetrics && len(metric.DeprecatedVersion) > 0 && !exporter.isExplicitlyEnabled(metric.Name)
}

func (exporter *BaseOpenStackExporter) AddMetric(name string, fn ListFunc, labels []string, deprecatedVersion string, constLabels prometheus.Labels) {
	if !exporter.IsMetricEnabled(name) {
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

// took from here:
// https://github.com/gophercloud/utils/blob/4c0f6d93d3a9b027a21d9206b6bdd09123de7a09/internal/util.go#L87
func pathOrContents(poc string) ([]byte, bool, error) {
	if len(poc) == 0 {
		return nil, false, nil
	}

	path := poc
	if path[0] == '~' {
		var err error
		path, err = homedir.Expand(path)
		if err != nil {
			return []byte(path), true, err
		}
	}

	if _, err := os.Stat(path); err == nil {
		contents, err := os.ReadFile(path)
		if err != nil {
			return contents, true, err
		}
		return contents, true, nil
	}

	return []byte(poc), false, nil
}

func NewExporter(name string, opts ExporterOptions, logger *slog.Logger) (OpenStackExporter, error) {
	var exporter OpenStackExporter
	var err error
	var transport http.RoundTripper
	var tlsConfig tls.Config

	optsv2 := clientconfigv2.ClientOpts{Cloud: opts.Cloud}

	config, err := clientconfigv2.GetCloudFromYAML(&optsv2)
	if err != nil {
		return nil, err
	}

	var configureTransport = false
	if !*config.Verify {
		logger.Info("SSL verification disabled on transport")
		tlsConfig.InsecureSkipVerify = true
		configureTransport = true
	} else if config.CACertFile != "" {
		certPool, err := additionalTLSTrust(config.CACertFile, logger)
		if err != nil {
			logger.Error("Failed to include additional certificates to ca-trust", "err", err)
		}
		tlsConfig.RootCAs = certPool
		configureTransport = true
	}

	// took from here:
	// https://github.com/gophercloud/utils/blob/4c0f6d93d3a9b027a21d9206b6bdd09123de7a09/internal/util.go#L65
	if config.ClientCertFile != "" && config.ClientKeyFile != "" {
		clientCert, _, err := pathOrContents(config.ClientCertFile)
		if err != nil {
			return nil, fmt.Errorf("error reading Client Cert: %s", err)
		}
		clientKey, _, err := pathOrContents(config.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("error reading Client Key: %s", err)
		}
		cert, err := tls.X509KeyPair(clientCert, clientKey)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		configureTransport = true
	}
	if configureTransport {
		transport = &http.Transport{TLSClientConfig: &tlsConfig}
	}

	if _, ok := os.LookupEnv("OS_DEBUG"); ok {
		if transport == nil {
			transport = http.DefaultTransport
		}

		transport = &clientutilsv2.RoundTripper{
			Rt:     transport,
			Logger: &clientutilsv2.DefaultLogger{},
		}
	}

	clientV2, err := NewServiceClientV2(name, &optsv2, transport, opts.EndpointType)
	if err != nil {
		return nil, err
	}

	uuidGenFunc := opts.UUIDGenFunc
	if uuidGenFunc == nil {
		uuidGenFunc = uuid.GenerateUUID
	}
	opts.UUIDGenFunc = uuidGenFunc

	exporterConfig := ExporterConfig{
		ExporterOptions: opts,
		ClientV2:        clientV2,
		ServiceName:     name,
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
