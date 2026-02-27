package exporters

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	gophercloudv2 "github.com/gophercloud/gophercloud/v2"
	openstackv2 "github.com/gophercloud/gophercloud/v2/openstack"
	gnocchiv2 "github.com/gophercloud/utils/v2/gnocchi"
	clientconfigv2 "github.com/gophercloud/utils/v2/openstack/clientconfig"
)

var serviceCatalogTypesByExporterService = map[string][]string{
	"network":         {"network"},
	"compute":         {"compute"},
	"image":           {"image"},
	"volume":          {"block-storage", "volume", "volumev2", "volumev3"},
	"identity":        {"identity"},
	"object-store":    {"object-store"},
	"load-balancer":   {"load-balancer"},
	"container-infra": {"container-infrastructure-management", "container-infra"},
	"dns":             {"dns"},
	"baremetal":       {"baremetal"},
	"gnocchi":         {"metric", "gnocchi"},
	"database":        {"database"},
	"orchestration":   {"orchestration"},
	"placement":       {"placement"},
	"sharev2":         {"shared-file-system", "sharev2"},
}

func AuthenticatedClientV2(opts *clientconfigv2.ClientOpts, transport http.RoundTripper) (*gophercloudv2.ProviderClient, error) {
	options, err := clientconfigv2.AuthOptions(opts)
	if err != nil {
		return nil, err
	}

	// Fixes #42
	options.AllowReauth = true

	client, err := openstackv2.NewClient(options.IdentityEndpoint)
	if err != nil {
		return nil, err
	}

	if transport != nil {
		if tr, ok := transport.(*http.Transport); ok {
			tr.Proxy = http.ProxyFromEnvironment
		}

		client.HTTPClient.Transport = transport
	}

	err = openstackv2.Authenticate(context.TODO(), client, *options)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func newAuthenticatedProviderClient(opts *clientconfigv2.ClientOpts, transport http.RoundTripper, endpointType string) (*gophercloudv2.ProviderClient, *clientconfigv2.Cloud, gophercloudv2.EndpointOpts, error) {
	cloud := new(clientconfigv2.Cloud)

	if opts == nil {
		opts = new(clientconfigv2.ClientOpts)
	}

	var cloudName string
	if opts.Cloud != "" {
		cloudName = opts.Cloud
	}

	envPrefix := "OS_"
	if opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if v := os.Getenv(envPrefix + "CLOUD"); v != "" {
		cloudName = v
	}

	if cloudName != "" {
		var err error
		cloud, err = clientconfigv2.GetCloudFromYAML(opts)
		if err != nil {
			return nil, nil, gophercloudv2.EndpointOpts{}, err
		}
	}

	pClient, err := AuthenticatedClientV2(opts, transport)
	if err != nil {
		return nil, nil, gophercloudv2.EndpointOpts{}, err
	}

	var region string
	if v := os.Getenv(envPrefix + "REGION_NAME"); v != "" {
		region = v
	}

	if v := cloud.RegionName; v != "" {
		region = v
	}

	if opts != nil {
		if v := opts.RegionName; v != "" {
			region = v
		}
	}

	eo := gophercloudv2.EndpointOpts{
		Region:       region,
		Availability: GetEndpointTypeV2(endpointType),
	}

	return pClient, cloud, eo, nil
}

func NewServiceClientV2(service string, opts *clientconfigv2.ClientOpts, transport http.RoundTripper, endpointType string) (*gophercloudv2.ServiceClient, error) {
	pClient, cloud, eo, err := newAuthenticatedProviderClient(opts, transport, endpointType)
	if err != nil {
		return nil, err
	}

	// Keep a map of the EndpointOpts for each service
	endpointOptsV2Mu.Lock()
	if endpointOptsV2 == nil {
		endpointOptsV2 = make(map[string]gophercloudv2.EndpointOpts)
	}
	endpointOptsV2[service] = eo
	endpointOptsV2Mu.Unlock()

	switch service {
	case "baremetal":
		return openstackv2.NewBareMetalV1(pClient, eo)
	case "compute":
		return openstackv2.NewComputeV2(pClient, eo)
	// NOTE: Intentionally disabled here: openstack-exporter has no "container" exporter.
	// case "container":
	// 	return openstackv2.NewContainerV1(pClient, eo)
	case "container-infra":
		return openstackv2.NewContainerInfraV1(pClient, eo)
	case "database":
		return openstackv2.NewDBV1(pClient, eo)
	case "dns":
		return openstackv2.NewDNSV2(pClient, eo)
	case "gnocchi":
		return gnocchiv2.NewGnocchiV1(pClient, eo)
	case "identity":
		identityVersion := "3"
		if v := cloud.IdentityAPIVersion; v != "" {
			identityVersion = v
		}

		switch identityVersion {
		case "v2", "2", "2.0":
			return openstackv2.NewIdentityV2(pClient, eo)
		case "v3", "3":
			return openstackv2.NewIdentityV3(pClient, eo)
		default:
			return nil, fmt.Errorf("invalid identity API version")
		}
	case "image":
		return openstackv2.NewImageV2(pClient, eo)
	case "load-balancer":
		return openstackv2.NewLoadBalancerV2(pClient, eo)
	case "network":
		return openstackv2.NewNetworkV2(pClient, eo)
	case "object-store":
		return openstackv2.NewObjectStorageV1(pClient, eo)
	case "orchestration":
		return openstackv2.NewOrchestrationV1(pClient, eo)
	case "placement":
		return openstackv2.NewPlacementV1(pClient, eo)
	case "sharev2":
		return openstackv2.NewSharedFileSystemV2(pClient, eo)
	case "volume":
		volumeVersion := "3"
		if v := cloud.VolumeAPIVersion; v != "" {
			volumeVersion = v
		}

		switch volumeVersion {
		case "v1", "1":
			return openstackv2.NewBlockStorageV1(pClient, eo)
		case "v2", "2":
			return openstackv2.NewBlockStorageV2(pClient, eo)
		case "v3", "3":
			return openstackv2.NewBlockStorageV3(pClient, eo)
		default:
			return nil, fmt.Errorf("invalid volume API version")
		}
	}

	return nil, fmt.Errorf("unable to create a service client for %s", service)
}

// GetEndpointType return openstack endpoints for configured type
func GetEndpointTypeV2(endpointType string) gophercloudv2.Availability {
	if endpointType == "internal" || endpointType == "internalURL" {
		return gophercloudv2.AvailabilityInternal
	}
	if endpointType == "admin" || endpointType == "adminURL" {
		return gophercloudv2.AvailabilityAdmin
	}
	return gophercloudv2.AvailabilityPublic
}

func additionalTLSTrust(caCertFile string, logger *slog.Logger) (*x509.CertPool, error) {
	// Get the SystemCertPool, continue with an empty pool on error
	trustedCAs, err := x509.SystemCertPool()
	if trustedCAs == nil {
		logger.Info("Creating a new empty SystemCertPool as we failed to load it from disk", "err", err)
		trustedCAs = x509.NewCertPool()
	}
	// check if string is not a path, but PEM contents such as: -----BEGIN CERTIFICATE-----
	if strings.HasPrefix(caCertFile, "---") {
		ok := trustedCAs.AppendCertsFromPEM(bytes.TrimSpace([]byte(caCertFile)))
		if !ok {
			return nil, fmt.Errorf("failed to add cert to trusted roots")
		}
	} else {
		pemFile, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, err
		}
		if ok := trustedCAs.AppendCertsFromPEM(bytes.TrimSpace(pemFile)); !ok {
			return nil, fmt.Errorf("error parsing CA Cert from: %s", caCertFile)
		}
	}
	return trustedCAs, nil
}

func mapStatus(mapping map[string]int, current string) int {
	v, ok := mapping[current]
	if !ok {
		return -1
	}

	return v
}

func AutodetectServicesFromCatalog(opts *clientconfigv2.ClientOpts, transport http.RoundTripper, endpointType string) ([]string, error) {
	providerClient, _, endpointOpts, err := newAuthenticatedProviderClient(opts, transport, endpointType)
	if err != nil {
		return nil, err
	}

	enabledServices := make([]string, 0, len(SupportedExporters))
	for _, service := range SupportedExporters {
		if !isServiceAvailable(providerClient, endpointOpts, service) {
			continue
		}
		enabledServices = append(enabledServices, service)
	}

	if len(enabledServices) == 0 {
		return nil, errors.New("no services autodetected")
	}

	return enabledServices, nil
}

func isServiceAvailable(providerClient *gophercloudv2.ProviderClient, endpointOpts gophercloudv2.EndpointOpts, service string) bool {
	serviceTypes, ok := serviceCatalogTypesByExporterService[service]
	if !ok {
		return false
	}

	for _, serviceType := range serviceTypes {
		eo := endpointOpts
		eo.ApplyDefaults(serviceType)
		endpoint, err := providerClient.EndpointLocator(eo)
		if err == nil && endpoint != "" {
			return true
		}
	}

	return false
}

func IsExporterNameValid(service string) bool {
	_, ok := serviceCatalogTypesByExporterService[service]
	return ok
}
