package exporters

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/gnocchi"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/grafana/regexp"
)

func AuthenticatedClient(opts *clientconfig.ClientOpts, transport *http.Transport) (*gophercloud.ProviderClient, error) {
	options, err := clientconfig.AuthOptions(opts)
	if err != nil {
		return nil, err
	}

	// Fixes #42
	options.AllowReauth = true

	client, err := openstack.NewClient(options.IdentityEndpoint)
	if err != nil {
		return nil, err
	}

	if transport != nil {
		transport.Proxy = http.ProxyFromEnvironment
		client.HTTPClient.Transport = transport
	}

	err = openstack.Authenticate(client, *options)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// NewServiceClient is a convenience function to get a new service client.
func NewServiceClient(service string, opts *clientconfig.ClientOpts, transport *http.Transport, endpointType string) (*gophercloud.ServiceClient, error) {
	cloud := new(clientconfig.Cloud)

	// If no opts were passed in, create an empty ClientOpts.
	if opts == nil {
		opts = new(clientconfig.ClientOpts)
	}

	// Determine if a clouds.yaml entry should be retrieved.
	// Start by figuring out the cloud name.
	// First check if one was explicitly specified in opts.
	var cloudName string
	if opts.Cloud != "" {
		cloudName = opts.Cloud
	}

	// Next see if a cloud name was specified as an environment variable.
	envPrefix := "OS_"
	if opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if v := os.Getenv(envPrefix + "CLOUD"); v != "" {
		cloudName = v
	}

	// If a cloud name was determined, try to look it up in clouds.yaml.
	if cloudName != "" {
		// Get the requested cloud.
		var err error
		cloud, err = clientconfig.GetCloudFromYAML(opts)
		if err != nil {
			return nil, err
		}
	}

	// Get a Provider Client
	pClient, err := AuthenticatedClient(opts, transport)
	if err != nil {
		return nil, err
	}

	// Determine the region to use.
	// First, check if the REGION_NAME environment variable is set.
	var region string
	if v := os.Getenv(envPrefix + "REGION_NAME"); v != "" {
		region = v
	}

	// Next, check if the cloud entry sets a region.
	if v := cloud.RegionName; v != "" {
		region = v
	}

	// Finally, see if one was specified in the ClientOpts.
	// If so, this takes precedence.
	if v := opts.RegionName; v != "" {
		region = v
	}

	eo := gophercloud.EndpointOpts{
		Region:       region,
		Availability: GetEndpointType(endpointType),
	}

	// Keep a map of the EndpointOpts for each service
	if endpointOpts == nil {
		endpointOpts = make(map[string]gophercloud.EndpointOpts)
	}
	endpointOpts[service] = eo

	switch service {
	case "baremetal":
		return openstack.NewBareMetalV1(pClient, eo)
	case "clustering":
		return openstack.NewClusteringV1(pClient, eo)
	case "compute":
		return openstack.NewComputeV2(pClient, eo)
	case "container":
		return openstack.NewContainerV1(pClient, eo)
	case "container-infra":
		return openstack.NewContainerInfraV1(pClient, eo)
	case "database":
		return openstack.NewDBV1(pClient, eo)
	case "dns":
		return openstack.NewDNSV2(pClient, eo)
	case "gnocchi":
		return gnocchi.NewGnocchiV1(pClient, eo)
	case "identity":
		identityVersion := "3"
		if v := cloud.IdentityAPIVersion; v != "" {
			identityVersion = v
		}

		switch identityVersion {
		case "v2", "2", "2.0":
			return openstack.NewIdentityV2(pClient, eo)
		case "v3", "3":
			return openstack.NewIdentityV3(pClient, eo)
		default:
			return nil, fmt.Errorf("invalid identity API version")
		}
	case "image":
		return openstack.NewImageServiceV2(pClient, eo)
	case "load-balancer":
		return openstack.NewLoadBalancerV2(pClient, eo)
	case "network":
		return openstack.NewNetworkV2(pClient, eo)
	case "object-store":
		return openstack.NewObjectStorageV1(pClient, eo)
	case "orchestration":
		return openstack.NewOrchestrationV1(pClient, eo)
	case "placement":
		return openstack.NewPlacementV1(pClient, eo)
	case "sharev2":
		return openstack.NewSharedFileSystemV2(pClient, eo)
	case "volume":
		volumeVersion := "3"
		if v := cloud.VolumeAPIVersion; v != "" {
			volumeVersion = v
		}

		switch volumeVersion {
		case "v1", "1":
			return openstack.NewBlockStorageV1(pClient, eo)
		case "v2", "2":
			return openstack.NewBlockStorageV2(pClient, eo)
		case "v3", "3":
			return openstack.NewBlockStorageV3(pClient, eo)
		default:
			return nil, fmt.Errorf("invalid volume API version")
		}
	}

	return nil, fmt.Errorf("unable to create a service client for %s", service)
}

// GetEndpointType return openstack endpoints for configured type
func GetEndpointType(endpointType string) gophercloud.Availability {
	if endpointType == "internal" || endpointType == "internalURL" {
		return gophercloud.AvailabilityInternal
	}
	if endpointType == "admin" || endpointType == "adminURL" {
		return gophercloud.AvailabilityAdmin
	}
	return gophercloud.AvailabilityPublic
}

// RemoveElements remove not needed elements
func RemoveElements(slice []string, drop []string) []string {
	res := []string{}
	for _, s := range slice {
		keep := true
		for _, d := range drop {
			if s == d {
				keep = false
				break
			}
		}
		if keep {
			res = append(res, s)
		}
	}
	return res
}

func additionalTLSTrust(caCertFile string, logger log.Logger) (*x509.CertPool, error) {
	// Get the SystemCertPool, continue with an empty pool on error
	trustedCAs, err := x509.SystemCertPool()
	if trustedCAs == nil {
		level.Info(logger).Log("msg", "Creating a new empty SystemCertPool as we failed to load it from disk", "err", err)
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

// SanitizeLabelName replaces anything that doesn't match
// client_label.LabelNameRE with an underscore.
//
// This has been copied from Prometheus sources at util/strutil/strconv.go
var invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
func SanitizeLabelName(name string) string {
	return invalidLabelCharRE.ReplaceAllString(name, "_")
}