package exporters

import (
	"context"
	"log/slog"

	"github.com/gophercloud/gophercloud/v2/openstack/keymanager/v1/containers"
	"github.com/gophercloud/gophercloud/v2/openstack/keymanager/v1/orders"
	"github.com/gophercloud/gophercloud/v2/openstack/keymanager/v1/secrets"
	"github.com/prometheus/client_golang/prometheus"
)

// Barbican only lists the resources of the token's project, so these counts
// are per project.

var knownOrderStatuses = []string{"PENDING", "ACTIVE", "ERROR"}

type BarbicanExporter struct {
	BaseOpenStackExporter
}

var defaultBarbicanMetrics = []Metric{
	{Name: "secrets", Fn: ListSecrets},
	{Name: "containers", Fn: ListSecretContainers},
	{Name: "orders", Labels: []string{"status"}, Fn: ListOrders},
}

func NewBarbicanExporter(config *ExporterConfig, logger *slog.Logger) (*BarbicanExporter, error) {
	exporter := BarbicanExporter{
		BaseOpenStackExporter{
			Name:           "barbican",
			ExporterConfig: *config,
			logger:         logger,
		},
	}

	for _, metric := range defaultBarbicanMetrics {
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}

	return &exporter, nil
}

func ListSecrets(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	allPages, err := secrets.List(exporter.ClientV2, secrets.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	allSecrets, err := secrets.ExtractSecrets(allPages)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["secrets"].Metric,
		prometheus.GaugeValue, float64(len(allSecrets)))

	return nil
}

func ListSecretContainers(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	allPages, err := containers.List(exporter.ClientV2, containers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	allContainers, err := containers.ExtractContainers(allPages)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["containers"].Metric,
		prometheus.GaugeValue, float64(len(allContainers)))

	return nil
}

func ListOrders(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	allPages, err := orders.List(exporter.ClientV2, orders.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	allOrders, err := orders.ExtractOrders(allPages)
	if err != nil {
		return err
	}

	orderStatusCounter := make(map[string]int, len(knownOrderStatuses))
	for _, status := range knownOrderStatuses {
		orderStatusCounter[status] = 0
	}
	for _, order := range allOrders {
		orderStatusCounter[order.Status]++
	}

	for status, count := range orderStatusCounter {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["orders"].Metric,
			prometheus.GaugeValue, float64(count), status)
	}

	return nil
}
