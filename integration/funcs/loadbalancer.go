package funcs

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/tools"
)

const (
	loadBalancerOperationTimeout = 15 * time.Minute
	loadBalancerPollInterval     = 5 * time.Second
)

// NewLoadBalancerClient returns an Octavia v2 client or fails the test.
func NewLoadBalancerClient(t *testing.T) *gophercloud.ServiceClient {
	t.Helper()

	client, err := clients.NewLoadBalancerV2Client()
	if err != nil {
		t.Fatalf("Failed to build load balancer client: %v", err)
	}
	return client
}

// MustCreateLoadBalancer creates an Octavia load balancer, waits for it to
// become ACTIVE, and registers cascade cleanup for it and all child resources.
func MustCreateLoadBalancer(t *testing.T, client *gophercloud.ServiceClient, subnetID string) *loadbalancers.LoadBalancer {
	t.Helper()

	name := tools.RandomString("ACPTTEST", 16)
	ctx, cancel := context.WithTimeout(t.Context(), loadBalancerOperationTimeout)
	defer cancel()

	loadBalancer, err := loadbalancers.Create(ctx, client, loadbalancers.CreateOpts{
		Name:         name,
		VipSubnetID:  subnetID,
		AdminStateUp: gophercloud.Enabled,
	}).Extract()
	if err != nil {
		t.Fatalf("Failed to create load balancer %s: %v", name, err)
	}

	t.Logf("Created load balancer %s (%s); waiting for ACTIVE", name, loadBalancer.ID)
	t.Cleanup(func() {
		deleteLoadBalancer(t, client, loadBalancer.ID)
	})

	return waitForLoadBalancerActive(t, ctx, client, loadBalancer.ID)
}

// MustCreateListener creates a TCP listener and waits for it to become ACTIVE.
// The parent load balancer's cascade cleanup removes the listener.
func MustCreateListener(t *testing.T, client *gophercloud.ServiceClient, loadBalancerID string) *listeners.Listener {
	t.Helper()

	name := tools.RandomString("ACPTTEST", 16)
	ctx, cancel := context.WithTimeout(t.Context(), loadBalancerOperationTimeout)
	defer cancel()

	listener, err := listeners.Create(ctx, client, listeners.CreateOpts{
		Name:           name,
		LoadbalancerID: loadBalancerID,
		Protocol:       listeners.ProtocolTCP,
		ProtocolPort:   8080,
		AdminStateUp:   gophercloud.Enabled,
	}).Extract()
	if err != nil {
		t.Fatalf("Failed to create listener %s on load balancer %s: %v", name, loadBalancerID, err)
	}

	t.Logf("Created listener %s (%s); waiting for ACTIVE", name, listener.ID)
	return waitForListenerActive(t, ctx, client, listener.ID)
}

func waitForLoadBalancerActive(t *testing.T, ctx context.Context, client *gophercloud.ServiceClient, id string) *loadbalancers.LoadBalancer {
	t.Helper()

	ticker := time.NewTicker(loadBalancerPollInterval)
	defer ticker.Stop()

	for {
		loadBalancer, err := loadbalancers.Get(ctx, client, id).Extract()
		if err != nil {
			t.Fatalf("Failed to get load balancer %s while waiting for ACTIVE: %v", id, err)
		}
		switch loadBalancer.ProvisioningStatus {
		case "ACTIVE":
			return loadBalancer
		case "ERROR":
			t.Fatalf("Load balancer %s entered ERROR state", id)
		}

		select {
		case <-ctx.Done():
			t.Fatalf("Timed out waiting for load balancer %s to become ACTIVE: %v", id, ctx.Err())
		case <-ticker.C:
		}
	}
}

func waitForListenerActive(t *testing.T, ctx context.Context, client *gophercloud.ServiceClient, id string) *listeners.Listener {
	t.Helper()

	ticker := time.NewTicker(loadBalancerPollInterval)
	defer ticker.Stop()

	for {
		listener, err := listeners.Get(ctx, client, id).Extract()
		if err != nil {
			t.Fatalf("Failed to get listener %s while waiting for ACTIVE: %v", id, err)
		}
		switch listener.ProvisioningStatus {
		case "ACTIVE":
			return listener
		case "ERROR":
			t.Fatalf("Listener %s entered ERROR state", id)
		}

		select {
		case <-ctx.Done():
			t.Fatalf("Timed out waiting for listener %s to become ACTIVE: %v", id, ctx.Err())
		case <-ticker.C:
		}
	}
}

func deleteLoadBalancer(t *testing.T, client *gophercloud.ServiceClient, id string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), loadBalancerOperationTimeout)
	defer cancel()

	t.Logf("Deleting load balancer %s with cascade", id)
	err := loadbalancers.Delete(ctx, client, id, loadbalancers.DeleteOpts{Cascade: true}).ExtractErr()
	if err != nil && !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Errorf("Failed to delete load balancer %s: %v", id, err)
		return
	}

	ticker := time.NewTicker(loadBalancerPollInterval)
	defer ticker.Stop()
	for {
		_, err = loadbalancers.Get(ctx, client, id).Extract()
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Logf("Deleted load balancer %s", id)
			return
		}
		if err != nil {
			t.Errorf("Failed to get load balancer %s while waiting for deletion: %v", id, err)
			return
		}

		select {
		case <-ctx.Done():
			t.Errorf("Timed out waiting for load balancer %s deletion: %v", id, ctx.Err())
			return
		case <-ticker.C:
		}
	}
}
