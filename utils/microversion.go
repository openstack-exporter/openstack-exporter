package utils

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/utils"
)

// SetupClientMicroversionV2 sets client microversion using environment variable or by some known good default
func SetupClientMicroversionV2(ctx context.Context, client *gophercloud.ServiceClient, envName, defaultLatest string, log *slog.Logger) error {
	lg := log.With("service_type", client.Type, "default_microversion", defaultLatest)

	// NOTE: utils.RequireMicroversion() from 2.9.0 do not work if the service returns multiple version endpoints, e.g. Nova.
	supportedVersions, err := utils.GetServiceVersions(ctx, client.ProviderClient, client.Endpoint, true)
	if err != nil {
		return fmt.Errorf("failed to get supported api versions: %w", err)
	}
	if len(supportedVersions) == 0 {
		return fmt.Errorf("microversions not supported by endpoint")
	}

	microversion, present := os.LookupEnv(envName)
	if !present {
		var found bool
		var prevMaxMajor, prevMaxMinor int
		for _, ver := range supportedVersions {
			if ok, _ := ver.IsSupported(defaultLatest); ok {
				microversion = defaultLatest
				found = true
				lg.Debug("Default microversion supported, set it")
				break
			} else {
				prevMaxMajor = max(prevMaxMajor, ver.MaxMajor)
				prevMaxMinor = max(prevMaxMinor, ver.MaxMinor)
			}
		}

		if !found && prevMaxMajor > 0 {
			microversion = fmt.Sprintf("%d.%d", prevMaxMajor, prevMaxMinor)
			lg.Warn("Default microversion not supported, set detected maximum available microversion", "detected_microversion", microversion)
		}

	} else {
		var found bool
		for _, ver := range supportedVersions {
			if ok, _ := ver.IsSupported(microversion); ok {
				found = true
				break
			}
		}

		if !found {
			lg.Error("Microvesion requested by env not supported", "env", envName, "requested_microversion", microversion)
			return fmt.Errorf("failed to require microversion: %s", microversion)
		}
	}

	lg.Debug("Set API microversion", "microversion", microversion)
	lg.Info("Set API microversion", "microversion", microversion, "endpoint", client.Endpoint)
	client.Microversion = microversion
	return nil
}
