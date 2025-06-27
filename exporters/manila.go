package exporters

import (
	"strconv"

	"github.com/go-kit/log"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/shares"
	"github.com/prometheus/client_golang/prometheus"
)

type ManilaExporter struct {
	BaseOpenStackExporter
}

var defaultManilaMetrics = []Metric{
	{Name: "shares_counter", Fn: CountShares},
	{Name: "share_gb", Labels: []string{"id", "name", "status", "availability_zone", "share_type", "share_proto", "share_type_name"}, Fn: nil},
	{Name: "share_status", Labels: []string{"id", "name", "status", "size", "share_type", "share_proto", "share_type_name"}, Fn: ListShareStatus},
	{Name: "share_status_counter", Labels: []string{"status"}, Fn: nil},
}

func NewManilaExporter(config *ExporterConfig, logger log.Logger) (*ManilaExporter, error) {
	exporter := ManilaExporter{
		BaseOpenStackExporter{

			Name:           "sharev2",
			ExporterConfig: *config,
			logger:         logger,
		},
	}

	for _, metric := range defaultManilaMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}

	return &exporter, nil
}

func CountShares(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {

	var allShares []shares.Share

	allPagesShares, err := shares.ListDetail(exporter.Client, shares.ListOpts{AllTenants: true}).AllPages()
	if err != nil {
		return err
	}

	allShares, err = shares.ExtractShares(allPagesShares)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["shares_counter"].Metric,
		prometheus.GaugeValue, float64(len(allShares)))

	// share_gb metrics
	for _, share := range allShares {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["share_gb"].Metric,
			prometheus.GaugeValue, float64(share.Size), share.ID, share.Name,
			share.Status, share.AvailabilityZone, share.ShareType, share.ShareProto, share.ShareTypeName)
	}

	share_status_counter := map[string]int{
		"creating":              0,
		"available":             0,
		"updating":              0,
		"migrating":             0,
		"migration_error":       0,
		"extending":             0,
		"deleting":              0,
		"shrinking":             0,
		"error":                 0,
		"error_deleting":        0,
		"shrinking_error":       0,
		"reverting_error":       0,
		"restoring":             0,
		"reverting":             0,
		"managing":              0,
		"unmanaging":            0,
		"reverting_to_snapshot": 0,
		"soft_deleting":         0,
		"inactive":              0,
	}

	for _, share := range allShares {
		share_status_counter[share.Status]++
	}

	// Share status counter metrics
	for status, count := range share_status_counter {
		ch <- prometheus.MustNewConstMetric(
			exporter.Metrics["share_status_counter"].Metric,
			prometheus.GaugeValue,
			float64(count),
			status)
	}

	return nil
}

func ListShareStatus(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {

	var allShares []shares.Share

	allPagesShares, err := shares.ListDetail(exporter.Client, shares.ListOpts{AllTenants: true}).AllPages()
	if err != nil {
		return err
	}

	allShares, err = shares.ExtractShares(allPagesShares)
	if err != nil {
		return err
	}

	// Share status metrics
	for _, share := range allShares {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["share_status"].Metric,
			prometheus.GaugeValue, float64(mapVolumeStatus(share.Status)), share.ID, share.Name,
			share.Status, strconv.Itoa(share.Size), share.ShareType, share.ShareProto, share.ShareTypeName)
	}
	return nil
}
