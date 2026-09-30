// Package clients contains functions for creating OpenStack service clients
// for use in acceptance tests. It also manages the required environment
// variables to run the tests.
package clients

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
)

// authTimeout is the maximum time allowed for authenticating against Keystone.
// It prevents tests from hanging indefinitely on connectivity problems.
const authTimeout = 30 * time.Second

// newAuthenticatedClient authenticates against Keystone using environment
// variables and returns a configured ProviderClient. A deadline is applied
// so that connectivity issues surface as a clear error rather than a CI timeout.
func newAuthenticatedClient() (*gophercloud.ProviderClient, error) {
	ao, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), authTimeout)
	defer cancel()

	client, err := openstack.AuthenticatedClient(ctx, ao)
	if err != nil {
		return nil, err
	}

	return configureDebug(client), nil
}

func regionOpts() gophercloud.EndpointOpts {
	return gophercloud.EndpointOpts{Region: os.Getenv("OS_REGION_NAME")}
}

func newServiceClient(newClient func(*gophercloud.ProviderClient, gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error)) (*gophercloud.ServiceClient, error) {
	client, err := newAuthenticatedClient()
	if err != nil {
		return nil, err
	}
	return newClient(client, regionOpts())
}

// NewBlockStorageV3Client returns a *ServiceClient for making calls
// to the OpenStack Block Storage v3 API. An error will be returned
// if authentication or client creation was not possible.
func NewBlockStorageV3Client() (*gophercloud.ServiceClient, error) {
	return newServiceClient(openstack.NewBlockStorageV3)
}

// NewComputeV2Client returns a *ServiceClient for making calls
// to the OpenStack Compute v2 API. An error will be returned
// if authentication or client creation was not possible.
func NewComputeV2Client() (*gophercloud.ServiceClient, error) {
	return newServiceClient(openstack.NewComputeV2)
}

// NewBareMetalV1Client returns a *ServiceClient for making calls
// to the OpenStack Bare Metal v1 API. An error will be returned
// if authentication or client creation was not possible.
func NewBareMetalV1Client() (*gophercloud.ServiceClient, error) {
	return newServiceClient(openstack.NewBareMetalV1)
}

// NewImageV2Client returns a *ServiceClient for making calls to the
// OpenStack Image v2 API. An error will be returned if authentication or
// client creation was not possible.
func NewImageV2Client() (*gophercloud.ServiceClient, error) {
	return newServiceClient(openstack.NewImageV2)
}

// NewNetworkV2Client returns a *ServiceClient for making calls to the
// OpenStack Networking v2 API. An error will be returned if authentication
// or client creation was not possible.
func NewNetworkV2Client() (*gophercloud.ServiceClient, error) {
	return newServiceClient(openstack.NewNetworkV2)
}

// configureDebug will configure the provider client to print the API
// requests and responses if OS_DEBUG is enabled. It uses the gophercloud
// utils RoundTripper which handles header redaction and JSON pretty-printing.
func configureDebug(client *gophercloud.ProviderClient) *gophercloud.ProviderClient {
	if os.Getenv("OS_DEBUG") != "" {
		client.HTTPClient = http.Client{
			Transport: newLoggingRoundTripper(),
		}
	}

	return client
}
