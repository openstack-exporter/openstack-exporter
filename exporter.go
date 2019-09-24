package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
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
	Name                 string
	Prefix               string
	Metrics              map[string]*prometheus.Desc
	Config               *Cloud
	AuthenticatedClient *gophercloud.ProviderClient
}

func (exporter *BaseOpenStackExporter) GetName() string {
	return fmt.Sprintf("%s_%s", exporter.Prefix, exporter.Name)
}

func (exporter *BaseOpenStackExporter) AddMetric(name string, labels []string, constLabels prometheus.Labels) {
	if exporter.Metrics == nil {
		exporter.Metrics = map[string]*prometheus.Desc{}
	}

	if constLabels == nil {
		constLabels = prometheus.Labels{}
	}

	constLabels["region"] = exporter.Config.Region

	if _, ok := exporter.Metrics[name]; !ok {
		log.Infof("Adding metric: %s to exporter: %s", name, exporter.Name)
		exporter.Metrics[name] = prometheus.NewDesc(
			prometheus.BuildFQName(exporter.GetName(), "", name),
			name, labels, constLabels)
	}
}

func NewExporter(name string, prefix string, config *Cloud) (OpenStackExporter, error) {
	var exporter OpenStackExporter
	var err error
	var newClient *gophercloud.ProviderClient

	clientOpts := new(clientconfig.ClientOpts)

	clientOpts.Cloud = "devstack-admin"

	_, err = clientconfig.GetCloudFromYAML(clientOpts)
	if err != nil {
		return nil, err
	}

	ao, err := clientconfig.AuthOptions(clientOpts)
	if err != nil {
		return nil, err
	}

	newClient, err = openstack.NewClient(ao.IdentityEndpoint)
	if err != nil {
		return nil, err
	}

	err = openstack.Authenticate(newClient, *ao)
	if err != nil {
		return nil, err
	}

	switch name {
	case "network":
		{
			exporter, err = NewNeutronExporter(newClient, prefix, config)
			if err != nil {
				return nil, err
			}
		}
	case "compute":
		{
			exporter, err = NewNovaExporter(newClient, prefix, config)
			if err != nil {
				return nil, err
			}
		}
	case "image":
		{
			exporter, err = NewGlanceExporter(newClient, prefix, config)
			if err != nil {
				return nil, err
			}
		}
	case "volumev3":
		{
			exporter, err = NewCinderExporter(newClient, prefix, config)
			if err != nil {
				return nil, err
			}
		}
	case "identity":
		{
			exporter, err = NewKeystoneExporter(newClient, prefix, config)
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
