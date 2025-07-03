package exporters

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gophercloud/gophercloud"
	gophercloudv2 "github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/utils/openstack/clientconfig"
	clientconfigv2 "github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/hashicorp/go-uuid"
	"github.com/prometheus/client_golang/prometheus"
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

type OpenStackExporter interface {
	prometheus.Collector

	GetName() string
	AddMetric(name string, fn ListFunc, labels []string, deprecatedVersion string, constLabels prometheus.Labels)
	MetricIsDisabled(name string) bool
}

func EnableExporter(service, prefix, cloud string, disabledMetrics []string, endpointType string, collectTime bool, disableSlowMetrics bool, disableDeprecatedMetrics bool, disableCinderAgentUUID bool, domainID string, tenantID string, uuidGenFunc func() (string, error), logger log.Logger) (*OpenStackExporter, error) {
	exporter, err := NewExporter(service, prefix, cloud, disabledMetrics, endpointType, collectTime, disableSlowMetrics, disableDeprecatedMetrics, disableCinderAgentUUID, domainID, tenantID, uuidGenFunc, logger)
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
	Client                   *gophercloud.ServiceClient
	ClientV2                 *gophercloudv2.ServiceClient
	Prefix                   string
	DisabledMetrics          []string
	CollectTime              bool
	UUIDGenFunc              func() (string, error)
	DisableSlowMetrics       bool
	DisableDeprecatedMetrics bool
	DisableCinderAgentUUID   bool
	DomainID                 string
	TenantID                 string
}

type BaseOpenStackExporter struct {
	ExporterConfig
	Name    string
	Metrics map[string]*PrometheusMetric
	logger  log.Logger
}

type ListFunc func(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error

var (
	endpointOpts   = make(map[string]gophercloud.EndpointOpts)
	endpointOptsMu sync.Mutex
)
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

func (exporter *BaseOpenStackExporter) RunCollection(metric *PrometheusMetric, metricName string, ch chan<- prometheus.Metric, logger log.Logger) error {
	level.Info(logger).Log("msg", "Collecting metrics for exporter", "exporter", exporter.GetName(), "metrics", metricName)
	now := time.Now()
	err := metric.Fn(exporter, ch)
	if err != nil {
		return fmt.Errorf("failed to collect metric: %s, error: %s", metricName, err)
	}

	level.Info(logger).Log("msg", "Collected metrics for exporter", "exporter", exporter.GetName(), "metrics", metricName)
	if exporter.CollectTime {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["openstack_metric_collect_seconds"].Metric, prometheus.GaugeValue, time.Since(now).Seconds(), metricName)
	}
	return nil
}

func (exporter *BaseOpenStackExporter) Collect(ch chan<- prometheus.Metric) {
	metricsDown := 0
	metricsCount := len(exporter.Metrics)

	for name, metric := range exporter.Metrics {
		if metric.Fn == nil {
			level.Debug(exporter.logger).Log("msg", "No function handler set for metric", "metric", name)
			metricsCount--
			continue
		}

		if err := exporter.RunCollection(metric, name, ch, exporter.logger); err != nil {
			level.Error(exporter.logger).Log("err", "Failed to collect metric for exporter", "exporter", exporter.Name, "error", err)
			metricsDown++
		}
	}

	//If all metrics collections fails for a given service, we'll flag it as down.
	if metricsDown >= metricsCount {
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
		level.Warn(exporter.logger).Log("msg", "metric has been disabled for exporter, not collecting metrics", "metric", name, "exporter", exporter.Name)
		return
	}

	if len(deprecatedVersion) > 0 {
		level.Warn(exporter.logger).Log("msg", "metric has been deprecated on exporter in version and it will be removed in next release", "metric", name, "exporter", exporter.Name, "version", deprecatedVersion)
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
		level.Info(exporter.logger).Log("msg", "Adding metric to exporter", "metric", name, "exporter", exporter.Name)
		exporter.Metrics[name] = &PrometheusMetric{
			Metric: prometheus.NewDesc(
				prometheus.BuildFQName(exporter.GetName(), "", name),
				name, labels, constLabels),
			Fn: fn,
		}
	}
}

func NewExporter(name, prefix, cloud string, disabledMetrics []string, endpointType string, collectTime bool, disableSlowMetrics bool, disableDeprecatedMetrics bool, disableCinderAgentUUID bool, domainID string, tenantID string, uuidGenFunc func() (string, error), logger log.Logger) (OpenStackExporter, error) {
	var exporter OpenStackExporter
	var err error
	var transport *http.Transport
	var tlsConfig tls.Config

	opts := clientconfig.ClientOpts{Cloud: cloud}
	optsv2 := clientconfigv2.ClientOpts{Cloud: cloud}

	config, err := clientconfig.GetCloudFromYAML(&opts)
	if err != nil {
		return nil, err
	}

	if !*config.Verify {
		level.Info(logger).Log("msg", "SSL verification disabled on transport")
		tlsConfig.InsecureSkipVerify = true
		transport = &http.Transport{TLSClientConfig: &tlsConfig}
	} else if config.CACertFile != "" {
		certPool, err := additionalTLSTrust(config.CACertFile, logger)
		if err != nil {
			level.Error(logger).Log("msg", "Failed to include additional certificates to ca-trust", "err", err)
		}
		tlsConfig.RootCAs = certPool
		transport = &http.Transport{TLSClientConfig: &tlsConfig}
	}

	client, err := NewServiceClient(name, &opts, transport, endpointType)
	if err != nil {
		return nil, err
	}

	clientV2, err := NewServiceClientV2(name, &optsv2, transport, endpointType)
	if err != nil {
		return nil, err
	}

	if uuidGenFunc == nil {
		uuidGenFunc = uuid.GenerateUUID
	}

	exporterConfig := ExporterConfig{
		Client:                   client,
		ClientV2:                 clientV2,
		Prefix:                   prefix,
		DisabledMetrics:          disabledMetrics,
		CollectTime:              collectTime,
		UUIDGenFunc:              uuidGenFunc,
		DisableSlowMetrics:       disableSlowMetrics,
		DisableDeprecatedMetrics: disableDeprecatedMetrics,
		DisableCinderAgentUUID:   disableCinderAgentUUID,
		DomainID:                 domainID,
		TenantID:                 tenantID,
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
