package exporters

import (
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/openstack-exporter/openstack-exporter/exporters/openstack"
)

func EnableExporter(name, service, prefix, cloud string, disabledMetrics []string,
	endpointType string, collectTime bool, disableSlowMetrics bool,
	disableDeprecatedMetrics bool, disableCinderAgentUUID bool, domainID string,
	uuidGenFunc func() (string, error), logger log.Logger) (*openstack.Exporter, error) {
	switch name {
	case "openstack":
		exporter, err := openstack.NewExporter(
			service, prefix, cloud, disabledMetrics, endpointType,
			collectTime, disableSlowMetrics, disableDeprecatedMetrics,
			disableCinderAgentUUID, domainID, uuidGenFunc, logger)
		if err != nil {
			return nil, err
		}
		return &exporter, nil
	case "opentelekomcloud":
		return nil, level.Error(logger).Log("err", "enabling exporter for api failed", "api", name, "error", "not implemented")
	default:
		return nil, level.Error(logger).Log("err", "enabling exporter for api failed", "api", name, "error", "not implemented")
	}
}
