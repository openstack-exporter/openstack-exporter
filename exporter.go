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

const (
	BYTE = 1 << (10 * iota)
	KILOBYTE
	MEGABYTE
	GIGABYTE
	TERABYTE
)

type OpenStackExporter interface {
	GetName() string
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
	var authMode identity.AuthMode

	credentials.URL = config.Auth.AuthURL
	credentials.ProjectDomain = config.Auth.ProjectDomainName
	credentials.UserDomain = config.Auth.UserDomainName
	credentials.Region = config.Region
	credentials.User = config.Auth.Username
	credentials.Secrets = config.Auth.Password
	credentials.TenantName = config.Auth.ProjectName

	if config.IdentityAPIVersion == "3" {
		authMode = identity.AuthUserPassV3
	} else {
		authMode = identity.AuthUserPass
	}

	tlsConfig, err := config.GetTLSConfig()
	if err != nil {
		return nil, err
	}

	if tlsConfig != nil {
		log.Infoln("using TLS configured SSL connection")
		newClient = client.NewClientTLSConfig(&credentials, authMode, nil, tlsConfig)
	} else {
		log.Infoln("using non TLS configured SSL connection")
		newClient = client.NewClient(&credentials, authMode, nil)
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
