package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/utils"
)

// SetupClientMicroversionV2 sets client microversion using environment variable or by some known good default
func SetupClientMicroversionV2(ctx context.Context, client *gophercloud.ServiceClient, envName, defaultLatest string) error {
	microversion, present := os.LookupEnv(envName)
	if !present {
		supportedVersions, err := utils.GetSupportedMicroversions(ctx, client)
		if err != nil {
			return err
		}

		if ok, _ := supportedVersions.IsSupported(defaultLatest); ok {
			microversion = defaultLatest
		} else {
			microversion = fmt.Sprintf("%d.%d", supportedVersions.MaxMajor, supportedVersions.MaxMinor)
		}
	} else {
		_, err := utils.RequireMicroversion(ctx, *client, microversion)
		if err != nil {
			return err
		}
	}

	client.Microversion = microversion
	return nil
}
