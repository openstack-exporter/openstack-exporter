package integration

import (
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/funcs"
)

func TestKeyManagerIntegration(t *testing.T) {
	clients.RequireLong(t)

	keyManagerClient, err := clients.NewKeyManagerV1Client()
	if err != nil {
		t.Fatalf("Failed to build key manager client: %v", err)
	}

	secretID, err := funcs.CreateSecret(t, keyManagerClient)
	if err != nil {
		t.Fatalf("Could not create test secret: %v", err)
	}
	defer funcs.DeleteSecret(t, keyManagerClient, secretID)

	_, cleanup, err := startOpenStackExporterForCloud("devstack-admin", []string{"key-manager"})
	if err != nil {
		t.Fatalf("Failed to start exporter: %v", err)
	}
	defer cleanup()
	metrics := scrapeLoggedMetrics(t, "")

	t.Run("openstack_barbican_up_metric", func(t *testing.T) {
		metrics.requireUp(t, "openstack_barbican_up")
	})

	t.Run("barbican_secrets_counted", func(t *testing.T) {
		metrics.requireMinValue(t, "openstack_barbican_secrets", nil, 1)
	})

	t.Run("barbican_order_statuses_present", func(t *testing.T) {
		metrics.requireMetric(t, "openstack_barbican_orders", labels{"status": "ERROR"})
	})
}
