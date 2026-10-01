package exporters

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/prometheus/client_golang/prometheus"
)

type ObjectStoreExporter struct {
	BaseOpenStackExporter
}

var defaultObjectStoreMetrics = []Metric{
	{Name: "objects", Labels: []string{"container_name", "project_id"}, Fn: ListContainers},
	{Name: "bytes", Labels: []string{"container_name", "project_id"}, Fn: nil},
}

func NewObjectStoreExporter(config *ExporterConfig, logger *slog.Logger) (*ObjectStoreExporter, error) {
	exporter := ObjectStoreExporter{
		BaseOpenStackExporter{
			Name:           "object_store",
			ExporterConfig: *config,
			logger:         logger,
		},
	}

	for _, metric := range defaultObjectStoreMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}

	return &exporter, nil
}

func ListContainers(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	projectIDs := []string{exporter.TenantID}
	if exporter.TenantID == "" {
		allProjects, err := GetProjects(ctx, exporter)
		if err != nil {
			return err
		}
		projectIDs = make([]string, len(allProjects))
		for i, project := range allProjects {
			projectIDs[i] = project.ID
		}
	}

	for _, projectID := range projectIDs {
		client := *exporter.ClientV2
		if exporter.TenantID == "" {
			endpoint, _, _ := strings.Cut(strings.TrimRight(client.Endpoint, "/"), "/v1/AUTH_")
			client.Endpoint = endpoint + "/v1/AUTH_" + projectID
		}
		err := containers.List(&client, containers.ListOpts{}).EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
			containerList, err := containers.ExtractInfo(page)
			if err != nil {
				return false, err
			}

			for _, c := range containerList {
				if !exporter.MetricIsDisabled("objects") {
					ch <- prometheus.MustNewConstMetric(exporter.Metrics["objects"].Metric,
						prometheus.GaugeValue, float64(c.Count), c.Name, projectID)
				}
				if !exporter.MetricIsDisabled("bytes") {
					ch <- prometheus.MustNewConstMetric(exporter.Metrics["bytes"].Metric,
						prometheus.GaugeValue, float64(c.Bytes), c.Name, projectID)
				}
			}
			return true, nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
