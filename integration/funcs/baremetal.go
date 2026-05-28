package funcs

import (
	"context"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/baremetal/v1/nodes"
	"github.com/openstack-exporter/openstack-exporter/integration/tools"
)

// DeleteNode deletes a bare metal node via its UUID.
func DeleteNode(t *testing.T, client *gophercloud.ServiceClient, node *nodes.Node) {
	// Force deletion of provisioned nodes requires maintenance mode.
	err := nodes.SetMaintenance(context.TODO(), client, node.UUID, nodes.MaintenanceOpts{
		Reason: "forced deletion",
	}).ExtractErr()
	if err != nil {
		t.Fatalf("Unable to move node %s into maintenance mode: %s", node.UUID, err)
	}

	err = nodes.Delete(context.TODO(), client, node.UUID).ExtractErr()
	if err != nil {
		t.Fatalf("Unable to delete node %s: %s", node.UUID, err)
	}

	t.Logf("Deleted server: %s", node.UUID)
}

// CreateFakeNode creates a node with fake-hardware.
func CreateFakeNode(t *testing.T, client *gophercloud.ServiceClient) (*nodes.Node, error) {
	name := tools.RandomString("ACPTTEST", 16)
	t.Logf("Attempting to create bare metal node: %s", name)

	node, err := nodes.Create(context.TODO(), client, nodes.CreateOpts{
		Name:            name,
		Driver:          "fake-hardware",
		BootInterface:   "fake",
		DeployInterface: "fake",
		DriverInfo: map[string]any{
			"ipmi_port":      "6230",
			"ipmi_username":  "admin",
			"deploy_kernel":  "http://172.22.0.1/images/tinyipa-stable-rocky.vmlinuz",
			"ipmi_address":   "192.168.122.1",
			"deploy_ramdisk": "http://172.22.0.1/images/tinyipa-stable-rocky.gz",
			"ipmi_password":  "admin",
		},
	}).Extract()

	return node, err
}

func ChangeProvisionStateAndWait(ctx context.Context, client *gophercloud.ServiceClient, node *nodes.Node,
	change nodes.ProvisionStateOpts, expectedState nodes.ProvisionState) (*nodes.Node, error) {
	err := nodes.ChangeProvisionState(ctx, client, node.UUID, change).ExtractErr()
	if err != nil {
		return node, err
	}

	err = nodes.WaitForProvisionState(ctx, client, node.UUID, expectedState)
	if err != nil {
		return node, err
	}

	return nodes.Get(ctx, client, node.UUID).Extract()
}

// DeployFakeNode deploys a node that uses fake-hardware.
func DeployFakeNode(t *testing.T, client *gophercloud.ServiceClient, node *nodes.Node) (*nodes.Node, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	currentState := node.ProvisionState

	if currentState == string(nodes.Enroll) {
		t.Logf("moving fake node %s to manageable", node.UUID)
		err := nodes.ChangeProvisionState(ctx, client, node.UUID, nodes.ProvisionStateOpts{
			Target: nodes.TargetManage,
		}).ExtractErr()
		if err != nil {
			return node, err
		}

		err = nodes.WaitForProvisionState(ctx, client, node.UUID, nodes.Manageable)
		if err != nil {
			return node, err
		}

		currentState = string(nodes.Manageable)
	}

	if currentState == string(nodes.Manageable) {
		t.Logf("moving fake node %s to available", node.UUID)
		err := nodes.ChangeProvisionState(ctx, client, node.UUID, nodes.ProvisionStateOpts{
			Target: nodes.TargetProvide,
		}).ExtractErr()
		if err != nil {
			return node, err
		}

		err = nodes.WaitForProvisionState(ctx, client, node.UUID, nodes.Available)
		if err != nil {
			return node, err
		}
	}

	t.Logf("deploying fake node %s", node.UUID)
	return ChangeProvisionStateAndWait(ctx, client, node, nodes.ProvisionStateOpts{
		Target: nodes.TargetActive,
	}, nodes.Active)
}
