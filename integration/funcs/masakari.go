package funcs

import (
	"context"
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/services"
	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/tools"
)

// masakariSegment represents a Masakari failover segment.
type masakariSegment struct {
	ID             int    `json:"id"`
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	RecoveryMethod string `json:"recovery_method"`
	ServiceType    string `json:"service_type"`
}

// masakariHost represents a Masakari host.
type masakariHost struct {
	ID                int    `json:"id"`
	UUID              string `json:"uuid"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	ControlAttributes string `json:"control_attributes"`
	OnMaintenance     bool   `json:"on_maintenance"`
	Reserved          bool   `json:"reserved"`
	FailoverSegmentID string `json:"failover_segment_id"`
}

// CreateMasakariSegment creates a failover segment with a random name.
// It returns the created segment. Use DeleteMasakariSegment to clean up.
func CreateMasakariSegment(t *testing.T, client *gophercloud.ServiceClient) (*masakariSegment, error) {
	name := tools.RandomString("ACCTEST", 16)
	t.Logf("Attempting to create Masakari segment: %s", name)

	body := map[string]any{
		"segment": map[string]any{
			"name":            name,
			"recovery_method": "auto",
			"service_type":    "compute",
		},
	}

	var result struct {
		Segment masakariSegment `json:"segment"`
	}

	url := client.ServiceURL("segments")
	if _, err := client.Post(context.TODO(), url, body, &result, nil); err != nil {
		return nil, err
	}

	t.Logf("Created Masakari segment: %s (uuid=%s)", result.Segment.Name, result.Segment.UUID)
	return &result.Segment, nil
}

// DeleteMasakariSegment deletes a failover segment by UUID.
func DeleteMasakariSegment(t *testing.T, client *gophercloud.ServiceClient, segmentUUID string) {
	url := client.ServiceURL("segments", segmentUUID)
	if _, err := client.Delete(context.TODO(), url, nil); err != nil {
		t.Fatalf("Unable to delete Masakari segment %s: %v", segmentUUID, err)
	}
	t.Logf("Deleted Masakari segment: %s", segmentUUID)
}

// CreateMasakariHost creates a host in the given failover segment.
// The host name must match a valid compute host name in Nova.
// It returns the created host. Use DeleteMasakariHost to clean up.
func CreateMasakariHost(t *testing.T, masakariClient *gophercloud.ServiceClient, segmentUUID string) (*masakariHost, error) {
	computeClient, err := clients.NewComputeV2Client()
	if err != nil {
		return nil, err
	}

	hostname, err := getComputeHostname(computeClient)
	if err != nil {
		return nil, err
	}

	t.Logf("Attempting to create Masakari host: %s in segment %s", hostname, segmentUUID)

	body := map[string]any{
		"host": map[string]any{
			"name":               hostname,
			"type":               "COMPUTE",
			"control_attributes": "SSH",
		},
	}

	var result struct {
		Host masakariHost `json:"host"`
	}

	url := masakariClient.ServiceURL("segments", segmentUUID, "hosts")
	if _, err := masakariClient.Post(context.TODO(), url, body, &result, nil); err != nil {
		return nil, err
	}

	t.Logf("Created Masakari host: %s (uuid=%s)", result.Host.Name, result.Host.UUID)
	return &result.Host, nil
}

// DeleteMasakariHost deletes a host from a failover segment.
func DeleteMasakariHost(t *testing.T, client *gophercloud.ServiceClient, segmentUUID, hostUUID string) {
	url := client.ServiceURL("segments", segmentUUID, "hosts", hostUUID)
	if _, err := client.Delete(context.TODO(), url, nil); err != nil {
		t.Fatalf("Unable to delete Masakari host %s: %v", hostUUID, err)
	}
	t.Logf("Deleted Masakari host: %s", hostUUID)
}

// getComputeHostname returns the hostname of the first available compute service.
func getComputeHostname(computeClient *gophercloud.ServiceClient) (string, error) {
	allPages, err := services.List(computeClient, services.ListOpts{}).AllPages(context.TODO())
	if err != nil {
		return "", err
	}

	allServices, err := services.ExtractServices(allPages)
	if err != nil {
		return "", err
	}

	for _, svc := range allServices {
		if svc.Binary == "nova-compute" && svc.Status == "enabled" {
			return svc.Host, nil
		}
	}

	return "", fmt.Errorf("no enabled nova-compute service found")
}
