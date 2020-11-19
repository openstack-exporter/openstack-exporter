package exporters

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/hashicorp/go-uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type Metric struct {
	Name   string
	Labels []string
	Fn     ListFunc
	Slow   bool
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
	AddMetric(name string, fn ListFunc, labels []string, constLabels prometheus.Labels)
	MetricIsDisabled(name string) bool
}

func EnableExporter(service, prefix, cloud string, disabledMetrics []string, endpointType string, collectTime bool, enableSlowMetrics bool, uuidGenFunc func() (string, error)) (*OpenStackExporter, error) {
	exporter, err := NewExporter(service, prefix, cloud, disabledMetrics, endpointType, collectTime, enableSlowMetrics, uuidGenFunc)
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
	Client             *gophercloud.ServiceClient
	Prefix             string
	DisabledMetrics    []string
	CollectTime        bool
	UUIDGenFunc        func() (string, error)
	DisableSlowMetrics bool
}

type BaseOpenStackExporter struct {
	ExporterConfig
	Name    string
	Metrics map[string]*PrometheusMetric
}

type ListFunc func(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error

var endpointOpts map[string]gophercloud.EndpointOpts

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

func (exporter *BaseOpenStackExporter) AddMetricCollectTime(collectTimeSeconds float64, metricName string, ch chan<- prometheus.Metric) {
	metricPromatheuslabels := prometheus.Labels{
		"openstack_service": exporter.GetName(),
		"openstack_metric":  metricName}
	metric := prometheus.NewDesc(
		"openstack_metric_collect_seconds",
		"Time needed to collect metric from OpenStack API",
		nil,
		metricPromatheuslabels)
	log.Debugf("Adding metric for collecting timings: %+s", metric)
	ch <- prometheus.MustNewConstMetric(metric, prometheus.GaugeValue, collectTimeSeconds)
}

func (exporter *BaseOpenStackExporter) RunCollection(metric *PrometheusMetric, metricName string, ch chan<- prometheus.Metric) error {
	log.Infof("Collecting metrics for exporter: %s, metric: %s", exporter.GetName(), metricName)
	now := time.Now()
	err := metric.Fn(exporter, ch)
	if err != nil {
		return fmt.Errorf("failed to collect metric: %s, error: %s", metricName, err)
	}

	log.Infof("Collected metrics for exporter: %s, metric: %s", exporter.GetName(), metricName)
	if exporter.CollectTime {
		exporter.AddMetricCollectTime(time.Since(now).Seconds(), metricName, ch)
	}
	return nil
}

func (exporter *BaseOpenStackExporter) Collect(ch chan<- prometheus.Metric) {
	serviceDown := 0

	for name, metric := range exporter.Metrics {
		if metric.Fn == nil {
			log.Debugf("No function handler set for metric: %s", name)
			continue
		}

		if err := exporter.RunCollection(metric, name, ch); err != nil {
			log.Errorf("Failed to collect metric for exporter: %s, error: %s", exporter.Name, err)
			serviceDown++
		}
	}

	if serviceDown > 0 {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["up"].Metric, prometheus.GaugeValue, 0)
	} else {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["up"].Metric, prometheus.GaugeValue, 1)
	}

}

func (exporter *BaseOpenStackExporter) isSlowMetric(metric *Metric) bool {
	return exporter.ExporterConfig.DisableSlowMetrics && metric.Slow
}

func (exporter *BaseOpenStackExporter) AddMetric(name string, fn ListFunc, labels []string, constLabels prometheus.Labels) {

	if exporter.MetricIsDisabled(name) {
		log.Warnf("metric: %s has been disabled on %s exporter, not collecting metrics", name, exporter.Name)
		return
	}

	if exporter.Metrics == nil {
		exporter.Metrics = make(map[string]*PrometheusMetric)
		exporter.Metrics["up"] = &PrometheusMetric{
			Metric: prometheus.NewDesc(
				prometheus.BuildFQName(exporter.GetName(), "", "up"),
				"up", nil, constLabels),
			Fn: nil,
		}
	}

	if constLabels == nil {
		constLabels = prometheus.Labels{}
	}

	// @TODO: get the region. constLabels["region"] = exporter.

	if _, ok := exporter.Metrics[name]; !ok {
		log.Infof("Adding metric: %s to exporter: %s", name, exporter.Name)
		exporter.Metrics[name] = &PrometheusMetric{
			Metric: prometheus.NewDesc(
				prometheus.BuildFQName(exporter.GetName(), "", name),
				name, labels, constLabels),
			Fn: fn,
		}
	}
}

func NewExporter(name, prefix, cloud string, disabledMetrics []string, endpointType string, collectTime bool, disableSlowMetrics bool, uuidGenFunc func() (string, error)) (OpenStackExporter, error) {
	var exporter OpenStackExporter
	var err error
	var transport *http.Transport

	opts := clientconfig.ClientOpts{Cloud: cloud}

	config, err := clientconfig.GetCloudFromYAML(&opts)
	if err != nil {
		return nil, err
	}

	if !*config.Verify {
		log.Infoln("SSL verification disabled on transport")
		tlsConfig := &tls.Config{InsecureSkipVerify: true}
		transport = &http.Transport{TLSClientConfig: tlsConfig}
	}

	if transport != nil {
		transport.Proxy = http.ProxyFromEnvironment
	}

	client, err := NewServiceClient(name, &opts, transport, endpointType)
	if err != nil {
		return nil, err
	}

	if uuidGenFunc == nil {
		uuidGenFunc = uuid.GenerateUUID
	}

	exporterConfig := ExporterConfig{
		Client:             client,
		Prefix:             prefix,
		DisabledMetrics:    disabledMetrics,
		CollectTime:        collectTime,
		UUIDGenFunc:        uuidGenFunc,
		DisableSlowMetrics: disableSlowMetrics,
	}

	switch name {
	case "network":
		{
			exporter, err = NewNeutronExporter(&exporterConfig)
			if err != nil {
				return nil, err
			}
		}
	case "compute":
		{
			exporter, err = NewNovaExporter(&exporterConfig)
			if err != nil {
				return nil, err
			}
		}
	case "image":
		{
			exporter, err = NewGlanceExporter(&exporterConfig)
			if err != nil {
				return nil, err
			}
		}
	case "volume":
		{
			exporter, err = NewCinderExporter(&exporterConfig)
			if err != nil {
				return nil, err
			}
		}
	case "identity":
		{
			exporter, err = NewKeystoneExporter(&exporterConfig)
			if err != nil {
				return nil, err
			}
		}
	case "object-store":
		{
			exporter, err = NewObjectStoreExporter(&exporterConfig)
			if err != nil {
				return nil, err
			}
		}
	case "load-balancer":
		{
			exporter, err = NewLoadbalancerExporter(&exporterConfig)
			if err != nil {
				return nil, err
			}
		}
	case "container-infra":
		{
			exporter, err = NewContainerInfraExporter(&exporterConfig)
			if err != nil {
				return nil, err
			}
		}
	case "dns":
		{
			exporter, err = NewDesignateExporter(&exporterConfig)
			if err != nil {
				return nil, err
			}
		}
	case "baremetal":
		{
			exporter, err = NewIronicExporter(&exporterConfig)
			if err != nil {
				return nil, err
			}
		}
	case "gnocchi":
		{
			exporter, err = NewGnocchiExporter(&exporterConfig)
			if err != nil {
				return nil, err
			}
		}
	default:
		{
			return nil, fmt.Errorf("couldn't find a handler for %s exporter", name)
		}
	}

	return exporter, nil
}
