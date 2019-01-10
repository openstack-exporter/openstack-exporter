package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"gopkg.in/niedbalski/goose.v3/client"
	"gopkg.in/niedbalski/goose.v3/identity"
)

type Metric struct {
	Name   string
	Labels []string
}

type OpenStackExporter interface {
	GetName() string
	GetMetrics() map[string]*prometheus.Desc
	AddMetric(name string, labels []string, constLabels prometheus.Labels)
	Describe(ch chan<- *prometheus.Desc)
	Collect(ch chan<- prometheus.Metric)
}

type BaseOpenStackExporter struct {
	Name    string
	Prefix  string
	Metrics map[string]*prometheus.Desc
	Config  *Cloud
}

func (exporter *BaseOpenStackExporter) GetName() string {
	return fmt.Sprintf("%s_%s", exporter.Prefix, exporter.Name)
}

func (exporter *BaseOpenStackExporter) GetMetrics() map[string]*prometheus.Desc {
	return exporter.Metrics
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
	var credentials identity.Credentials
	var newClient client.AuthenticatingClient

	credentials.URL = config.Auth.AuthURL
	credentials.ProjectDomain = config.Auth.ProjectDomainName
	credentials.UserDomain = config.Auth.UserDomainName
	credentials.Region = config.Region
	credentials.User = config.Auth.Username
	credentials.Secrets = config.Auth.Password
	credentials.TenantName = config.Auth.ProjectName

	if config.IdentityAPIVersion == "3" {
		newClient = client.NewClient(&credentials, identity.AuthUserPassV3, nil)
	} else {
		newClient = client.NewClient(&credentials, identity.AuthUserPass, nil)
	}

	newClient.SetRequiredServiceTypes([]string{name})
	newClient.Authenticate()

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
	case "volume":
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
