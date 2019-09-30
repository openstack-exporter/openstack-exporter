package exporters

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type Metric struct {
	Name   string
	Labels []string
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
	GetName() string
	AddMetric(name string, labels []string, constLabels prometheus.Labels)
	Describe(ch chan<- *prometheus.Desc)
	Collect(ch chan<- prometheus.Metric)
	RefreshClient() error
}

type BaseOpenStackExporter struct {
	Name    string
	Prefix  string
	Metrics map[string]*prometheus.Desc
	Client  *gophercloud.ServiceClient
}

func (exporter *BaseOpenStackExporter) GetName() string {
	return fmt.Sprintf("%s_%s", exporter.Prefix, exporter.Name)
}

func (exporter *BaseOpenStackExporter) RefreshClient() error {
	log.Debugln("Refreshing auth client in case token has expired")
	if err := exporter.Client.Reauthenticate(exporter.Client.Token()); err != nil {
		return err
	}
	return nil
}

func (exporter *BaseOpenStackExporter) AddMetric(name string, labels []string, constLabels prometheus.Labels) {
	if exporter.Metrics == nil {
		exporter.Metrics = map[string]*prometheus.Desc{}
	}

	if constLabels == nil {
		constLabels = prometheus.Labels{}
	}

	// @TODO: get the region. constLabels["region"] = exporter.

	if _, ok := exporter.Metrics[name]; !ok {
		log.Infof("Adding metric: %s to exporter: %s", name, exporter.Name)
		exporter.Metrics[name] = prometheus.NewDesc(
			prometheus.BuildFQName(exporter.GetName(), "", name),
			name, labels, constLabels)
	}
}

func NewExporter(name, prefix, cloud string) (OpenStackExporter, error) {
	var exporter OpenStackExporter
	var err error

	client, err := clientconfig.NewServiceClient(name, &clientconfig.ClientOpts{Cloud: cloud})
	if err != nil {
		return nil, err
	}

	switch name {
	case "network":
		{
			exporter, err = NewNeutronExporter(client, prefix)
			if err != nil {
				return nil, err
			}
		}
	case "compute":
		{
			exporter, err = NewNovaExporter(client, prefix)
			if err != nil {
				return nil, err
			}
		}
	case "image":
		{
			exporter, err = NewGlanceExporter(client, prefix)
			if err != nil {
				return nil, err
			}
		}
	case "volume":
		{
			exporter, err = NewCinderExporter(client, prefix)
			if err != nil {
				return nil, err
			}
		}
	case "identity":
		{
			exporter, err = NewKeystoneExporter(client, prefix)
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
