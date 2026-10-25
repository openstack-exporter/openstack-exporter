package exporters

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/instanceha/v1/hosts"
	"github.com/gophercloud/gophercloud/v2/openstack/instanceha/v1/notifications"
	"github.com/gophercloud/gophercloud/v2/openstack/instanceha/v1/segments"
	"github.com/gophercloud/gophercloud/v2/openstack/instanceha/v1/vmoves"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	masakariLookback = kingpin.Flag("masakari-notification-lookback", "History window for Masakari notifications; 0 includes all retained history. New and running notifications are always included.").Default("24h").Duration()
	masakariVMoves   = kingpin.Flag("masakari-collect-vmoves", "Collect Masakari VM moves (requires API 1.3).").Default("true").Bool()
	masakariTimeout  = kingpin.Flag("masakari-scrape-timeout", "Timeout for the complete Masakari API collection.").Default("30s").Duration()
)

type MasakariExporter struct {
	BaseOpenStackExporter
	lookback      time.Duration
	timeout       time.Duration
	collectVMoves bool
}

var masakariNotificationLabels = []string{"notification_uuid", "source_host_uuid", "type", "status"}
var masakariVMoveLabels = []string{"id", "notification_uuid", "instance_uuid", "source_host", "dest_host", "type", "status"}
var masakariHostLabels = []string{"id", "uuid", "hostname", "failover_segment_id", "failover_segment_name", "type", "control_attributes"}

var defaultMasakariMetrics = []Metric{
	{Name: "segments"},
	{Name: "segment", Labels: []string{"uuid", "name", "service_type", "recovery_method"}},
	{Name: "segment_enabled", Labels: []string{"uuid", "name"}},
	{Name: "hosts", Labels: []string{"segment_uuid"}},
	{Name: "reserved_hosts_available", Labels: []string{"segment_uuid"}},
	{Name: "host", Labels: masakariHostLabels},
	{Name: "host_on_maintenance", Labels: masakariHostLabels},
	{Name: "host_reserved", Labels: masakariHostLabels},
	{Name: "notifications", Labels: []string{"type", "status"}},
	{Name: "notification", Labels: masakariNotificationLabels},
	{Name: "notification_generated_timestamp_seconds", Labels: masakariNotificationLabels},
	{Name: "notification_updated_timestamp_seconds", Labels: masakariNotificationLabels},
	{Name: "notification_oldest_pending_age_seconds", Labels: []string{"type"}},
	{Name: "vmoves", Labels: []string{"type", "status"}, Slow: true},
	{Name: "vmove", Labels: masakariVMoveLabels, Slow: true},
	{Name: "vmove_start_timestamp_seconds", Labels: masakariVMoveLabels, Slow: true},
	{Name: "vmove_end_timestamp_seconds", Labels: masakariVMoveLabels, Slow: true},
	{Name: "vmove_duration_seconds", Labels: masakariVMoveLabels, Slow: true},
	{Name: "collection_success", Labels: []string{"collector"}},
}

func NewMasakariExporter(config *ExporterConfig, logger *slog.Logger) (*MasakariExporter, error) {
	if *masakariLookback < 0 || *masakariTimeout <= 0 {
		return nil, fmt.Errorf("Masakari lookback must be nonnegative and scrape timeout must be positive")
	}
	exporter := &MasakariExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{ExporterConfig: *config, Name: "masakari", logger: logger},
		lookback:              *masakariLookback, timeout: *masakariTimeout,
		collectVMoves: *masakariVMoves && !config.DisableSlowMetrics,
	}
	for _, metric := range defaultMasakariMetrics {
		if exporter.isSlowMetric(&metric) || (metric.Slow && !exporter.collectVMoves) {
			continue
		}
		exporter.AddMetric(metric.Name, nil, metric.Labels, "", nil)
	}
	return exporter, nil
}

func (exporter *MasakariExporter) emit(ch chan<- prometheus.Metric, name string, value float64, labels ...string) {
	if metric, ok := exporter.Metrics[name]; ok {
		ch <- prometheus.MustNewConstMetric(metric.Metric, prometheus.GaugeValue, value, labels...)
	}
}

func (exporter *MasakariExporter) enabled(names ...string) bool {
	for _, name := range names {
		if _, ok := exporter.Metrics[name]; ok {
			return true
		}
	}
	return false
}

func (exporter *MasakariExporter) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), exporter.timeout)
	defer cancel()
	ok := true
	report := func(group string, started time.Time, err error) {
		value := 1.0
		if err != nil {
			ok = false
			value = 0
			exporter.logger.Error("Failed to collect Masakari metrics", "collector", group, "err", err)
		}
		exporter.emit(ch, "collection_success", value, group)
		if exporter.CollectTime {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["openstack_metric_collect_seconds"].Metric, prometheus.GaugeValue, time.Since(started).Seconds(), group)
		}
	}
	if exporter.enabled("segments", "segment", "segment_enabled", "hosts", "reserved_hosts_available", "host", "host_on_maintenance", "host_reserved") {
		started := time.Now()
		report("inventory", started, exporter.collectInventory(ctx, ch))
	}
	if exporter.enabled("notifications", "notification", "notification_generated_timestamp_seconds", "notification_updated_timestamp_seconds", "notification_oldest_pending_age_seconds", "vmoves", "vmove", "vmove_start_timestamp_seconds", "vmove_end_timestamp_seconds", "vmove_duration_seconds") {
		started := time.Now()
		items, err := exporter.listNotifications(ctx, time.Now())
		report("notifications", started, err)
		if err == nil {
			exporter.emitNotifications(ch, items, time.Now())
			if exporter.collectVMoves && exporter.enabled("vmoves", "vmove", "vmove_start_timestamp_seconds", "vmove_end_timestamp_seconds", "vmove_duration_seconds") {
				started = time.Now()
				report("vmoves", started, exporter.collectMoves(ctx, ch, items))
			}
		}
	}
	if ok {
		exporter.emit(ch, "up", 1)
	} else {
		exporter.emit(ch, "up", 0)
	}
}

func (exporter *MasakariExporter) collectInventory(ctx context.Context, ch chan<- prometheus.Metric) error {
	pages, err := segments.List(exporter.ClientV2, segments.ListOpts{Limit: 1000, SortKey: "id", SortDir: "asc"}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := segments.ExtractSegments(pages)
	if err != nil {
		return err
	}
	allHosts := make(map[string][]hosts.Host)
	needHosts := exporter.enabled("hosts", "reserved_hosts_available", "host", "host_on_maintenance", "host_reserved")
	for _, segment := range items {
		if !needHosts {
			break
		}
		pages, err := hosts.List(exporter.ClientV2, segment.UUID, hosts.ListOpts{Limit: 1000, SortKey: "id", SortDir: "asc"}).AllPages(ctx)
		if err != nil {
			return err
		}
		allHosts[segment.UUID], err = hosts.ExtractHosts(pages)
		if err != nil {
			return err
		}
	}
	exporter.emit(ch, "segments", float64(len(items)))
	for _, segment := range items {
		exporter.emit(ch, "segment", 1, segment.UUID, segment.Name, segment.ServiceType, segment.RecoveryMethod)
		if segment.Enabled != nil {
			exporter.emit(ch, "segment_enabled", boolValue(*segment.Enabled), segment.UUID, segment.Name)
		}
		if !needHosts {
			continue
		}
		hostItems := allHosts[segment.UUID]
		exporter.emit(ch, "hosts", float64(len(hostItems)), segment.UUID)
		available := 0
		for _, host := range hostItems {
			labels := []string{strconv.Itoa(host.ID), host.UUID, host.Name, segment.UUID, segment.Name, host.Type, host.ControlAttributes}
			exporter.emit(ch, "host", 1, labels...)
			exporter.emit(ch, "host_on_maintenance", boolValue(host.OnMaintenance), labels...)
			exporter.emit(ch, "host_reserved", boolValue(host.Reserved), labels...)
			if host.Reserved && !host.OnMaintenance {
				available++
			}
		}
		exporter.emit(ch, "reserved_hosts_available", float64(available), segment.UUID)
	}
	return nil
}

func (exporter *MasakariExporter) listNotifications(ctx context.Context, now time.Time) ([]notifications.Notification, error) {
	opts := notifications.ListOpts{Limit: 1000, SortKey: "id", SortDir: "asc"}
	if exporter.lookback > 0 {
		opts.GeneratedSince = now.Add(-exporter.lookback).UTC().Format(time.RFC3339)
	}
	queries := []notifications.ListOpts{opts}
	if exporter.lookback > 0 {
		for _, status := range []string{"new", "running"} {
			queries = append(queries, notifications.ListOpts{Limit: 1000, SortKey: "id", SortDir: "asc", Status: status})
		}
	}
	var result []notifications.Notification
	seen := make(map[string]bool)
	for _, query := range queries {
		pages, err := notifications.List(exporter.ClientV2, query).AllPages(ctx)
		if err != nil {
			return nil, err
		}
		items, err := notifications.ExtractNotifications(pages)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if !seen[item.UUID] {
				result = append(result, item)
				seen[item.UUID] = true
			}
		}
	}
	return result, nil
}

func (exporter *MasakariExporter) emitNotifications(ch chan<- prometheus.Metric, items []notifications.Notification, now time.Time) {
	counts := make(map[[2]string]int)
	oldest := make(map[string]float64)
	for _, typ := range []string{"COMPUTE_HOST", "VM", "PROCESS"} {
		for _, status := range []string{"new", "running", "error", "failed", "ignored", "finished"} {
			counts[[2]string{typ, status}] = 0
		}
		oldest[typ] = 0
	}
	for _, item := range items {
		counts[[2]string{item.Type, item.Status}]++
		labels := []string{item.UUID, item.SourceHostUUID, item.Type, item.Status}
		exporter.emit(ch, "notification", 1, labels...)
		generated := time.Time(item.GeneratedTime)
		if !generated.IsZero() {
			exporter.emit(ch, "notification_generated_timestamp_seconds", float64(generated.Unix()), labels...)
			if item.Status == "new" || item.Status == "running" {
				age := now.Sub(generated).Seconds()
				if age > oldest[item.Type] {
					oldest[item.Type] = age
				}
			}
		}
		updated := time.Time(item.UpdatedAt)
		if !updated.IsZero() {
			exporter.emit(ch, "notification_updated_timestamp_seconds", float64(updated.Unix()), labels...)
		}
	}
	for key, count := range counts {
		exporter.emit(ch, "notifications", float64(count), key[0], key[1])
	}
	for typ, age := range oldest {
		exporter.emit(ch, "notification_oldest_pending_age_seconds", age, typ)
	}
}

func (exporter *MasakariExporter) collectMoves(ctx context.Context, ch chan<- prometheus.Metric, items []notifications.Notification) error {
	client := *exporter.ClientV2
	client.Microversion = "1.3"
	var moves []vmoves.VMove
	for _, item := range items {
		if item.Type != "COMPUTE_HOST" {
			continue
		}
		pages, err := vmoves.List(&client, item.UUID, vmoves.ListOpts{Limit: 1000, SortKey: "id", SortDir: "asc"}).AllPages(ctx)
		if err != nil {
			return err
		}
		batch, err := vmoves.ExtractVMoves(pages)
		if err != nil {
			return err
		}
		moves = append(moves, batch...)
	}
	counts := make(map[[2]string]int)
	for _, typ := range []string{"evacuation", "migration", "live_migration"} {
		for _, status := range []string{"pending", "ongoing", "ignored", "failed", "succeeded"} {
			counts[[2]string{typ, status}] = 0
		}
	}
	for _, move := range moves {
		counts[[2]string{move.Type, move.Status}]++
		labels := []string{strconv.Itoa(move.ID), move.NotificationUUID, move.InstanceUUID, move.SourceHost, move.DestHost, move.Type, move.Status}
		exporter.emit(ch, "vmove", 1, labels...)
		start, end := time.Time(move.StartTime), time.Time(move.EndTime)
		if !start.IsZero() {
			exporter.emit(ch, "vmove_start_timestamp_seconds", float64(start.Unix()), labels...)
		}
		if !end.IsZero() {
			exporter.emit(ch, "vmove_end_timestamp_seconds", float64(end.Unix()), labels...)
		}
		if !start.IsZero() && !end.IsZero() && !end.Before(start) {
			exporter.emit(ch, "vmove_duration_seconds", end.Sub(start).Seconds(), labels...)
		}
	}
	for key, count := range counts {
		exporter.emit(ch, "vmoves", float64(count), key[0], key[1])
	}
	return nil
}

func boolValue(value bool) float64 {
	if value {
		return 1
	}
	return 0
}
