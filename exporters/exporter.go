package exporters

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type Metric struct {
	Name   string
	Labels []string
	Fn     ListFunc
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

func EnableExporter(service, prefix, cloud string, disabledMetrics []string, endpointType string) (*OpenStackExporter, error) {
	exporter, err := NewExporter(service, prefix, cloud, disabledMetrics, endpointType)
	if err != nil {
		return nil, err
	}
	prometheus.MustRegister(exporter)
	return &exporter, nil
}

type PrometheusMetric struct {
	Metric *prometheus.Desc
	Fn     ListFunc
}

type BaseOpenStackExporter struct {
	Name            string
	Prefix          string
	Metrics         map[string]*PrometheusMetric
	Client          *gophercloud.ServiceClient
	DisabledMetrics []string
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

func (exporter *BaseOpenStackExporter) Collect(ch chan<- prometheus.Metric) {
	serviceUp := true

	for name, metric := range exporter.Metrics {
		log.Infof("Collecting metrics for exporter: %s, metric: %s", exporter.GetName(), name)
		if metric.Fn == nil {
			log.Debugf("No function handler set for metric: %s", name)
			continue
		}

		err := metric.Fn(exporter, ch)
		if err != nil {
			log.Errorln(err)
			serviceUp = false
		}
	}

	if serviceUp {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["up"].Metric, prometheus.GaugeValue, 1)
	} else {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["up"].Metric, prometheus.GaugeValue, 0)
	}
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

func NewExporter(name, prefix, cloud string, disabledMetrics []string, endpointType string) (OpenStackExporter, error) {
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

	client, err := NewServiceClient(name, &opts, transport, endpointType)
	if err != nil {
		return nil, err
	}

	switch name {
	case "network":
		{
			exporter, err = NewNeutronExporter(client, prefix, disabledMetrics)
			if err != nil {
				return nil, err
			}
		}
	case "compute":
		{
			exporter, err = NewNovaExporter(client, prefix, disabledMetrics)
			if err != nil {
				return nil, err
			}
		}
	case "image":
		{
			exporter, err = NewGlanceExporter(client, prefix, disabledMetrics)
			if err != nil {
				return nil, err
			}
		}
	case "volume":
		{
			exporter, err = NewCinderExporter(client, prefix, disabledMetrics)
			if err != nil {
				return nil, err
			}
		}
	case "identity":
		{
			exporter, err = NewKeystoneExporter(client, prefix, disabledMetrics)
			if err != nil {
				return nil, err
			}
		}
	case "object-store":
		{
			exporter, err = NewObjectStoreExporter(client, prefix, disabledMetrics)
			if err != nil {
				return nil, err
			}
		}
	case "load-balancer":
		{
			exporter, err = NewLoadbalancerExporter(client, prefix, disabledMetrics)
			if err != nil {
				return nil, err
			}
		}
	case "container-infra":
		{
			exporter, err = NewContainerInfraExporter(client, prefix, disabledMetrics)
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
