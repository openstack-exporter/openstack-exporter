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
	AuthenticatingClient client.AuthenticatingClient
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

	// Change service name to the v3 version of cinder/block storage API.
	// Part of fix for: https://github.com/openstack-exporter/openstack-exporter/issues/1
	if name == "volume" {
		name = "volumev3"
	}

	newClient.SetRequiredServiceTypes([]string{name})

	if err := newClient.Authenticate(); err != nil {
		return nil, fmt.Errorf("error when authenticating: %s", err)
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
