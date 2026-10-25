package exporters

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	openstack "github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

const masakariTestNotification = `{"id":11,"notification_uuid":"event","source_host_uuid":"host","type":"COMPUTE_HOST","status":"finished","generated_time":"2026-10-25T12:00:00.123456","updated_at":null}`
const masakariOldNotification = `{"id":12,"notification_uuid":"old","source_host_uuid":"host","type":"VM","status":"running","generated_time":"2020-01-01T00:00:00","updated_at":null}`

func masakariTestExporter(t *testing.T, handler http.HandlerFunc, disabled []string, slow bool) (*MasakariExporter, *prometheus.Registry) {
	t.Helper()
	previousLookback, previousTimeout, previousVMoves := *masakariLookback, *masakariTimeout, *masakariVMoves
	*masakariLookback, *masakariTimeout, *masakariVMoves = 24*time.Hour, 30*time.Second, true
	t.Cleanup(func() {
		*masakariLookback, *masakariTimeout, *masakariVMoves = previousLookback, previousTimeout, previousVMoves
	})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	exporter, err := NewMasakariExporter(&ExporterConfig{
		ClientV2: &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{}, Endpoint: server.URL + "/v1/", Type: "instance-ha"},
		Prefix:   "openstack", DisabledMetrics: disabled, DisableSlowMetrics: slow,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	registry := prometheus.NewPedanticRegistry()
	if err := registry.Register(exporter); err != nil {
		t.Fatal(err)
	}
	return exporter, registry
}

func masakariGather(t *testing.T, registry *prometheus.Registry) map[string]*dto.MetricFamily {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	result := make(map[string]*dto.MetricFamily)
	for _, family := range families {
		result[strings.TrimPrefix(family.GetName(), "openstack_masakari_")] = family
	}
	return result
}

func masakariSample(t *testing.T, families map[string]*dto.MetricFamily, name string, labels map[string]string, expected float64) {
	t.Helper()
	for _, sample := range families[name].GetMetric() {
		actual := make(map[string]string)
		for _, label := range sample.GetLabel() {
			actual[label.GetName()] = label.GetValue()
		}
		match := true
		for key, value := range labels {
			if actual[key] != value {
				match = false
			}
		}
		if match {
			if sample.GetGauge().GetValue() != expected {
				t.Fatalf("%s %v: expected %g, got %g", name, labels, expected, sample.GetGauge().GetValue())
			}
			return
		}
	}
	t.Fatalf("missing sample %s %v", name, labels)
}

func masakariFixtureHandler(t *testing.T, fail string, code int) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Path == fail {
			w.WriteHeader(code)
			fmt.Fprint(w, "{}")
			return
		}
		key := ""
		switch req.URL.Path {
		case "/v1/segments":
			key = "segments"
		case "/v1/segments/segment/hosts":
			key = "hosts"
		case "/v1/notifications":
			key = "notifications"
		case "/v1/notifications/event/vmoves":
			key = "vmoves"
		default:
			t.Errorf("unexpected request %s", req.URL)
			w.WriteHeader(404)
			return
		}
		if req.URL.Query().Get("marker") != "" {
			fmt.Fprintf(w, `{"%s":[]}`, key)
			return
		}
		switch key {
		case "segments":
			fmt.Fprint(w, `{"segments":[{"id":1,"uuid":"segment","name":"compute","service_type":"COMPUTE","recovery_method":"reserved_host","enabled":false}]}`)
		case "hosts":
			fmt.Fprint(w, `{"hosts":[{"id":2,"uuid":"host","name":"compute-1","type":"COMPUTE_HOST","control_attributes":"SSH","reserved":true,"on_maintenance":false},{"id":3,"uuid":"maintenance","name":"compute-2","reserved":true,"on_maintenance":true}]}`)
		case "notifications":
			if req.URL.Query().Get("status") == "running" {
				fmt.Fprint(w, `{"notifications":[`+masakariOldNotification+`]}`)
			} else if req.URL.Query().Get("status") == "new" {
				fmt.Fprint(w, `{"notifications":[]}`)
			} else {
				if req.URL.Query().Get("generated-since") == "" {
					t.Error("missing history filter")
				}
				fmt.Fprint(w, `{"notifications":[`+masakariTestNotification+`,`+masakariOldNotification+`]}`)
			}
		case "vmoves":
			if req.Header.Get("OpenStack-API-Version") != "instance-ha 1.3" {
				t.Error("missing VM move microversion")
			}
			fmt.Fprint(w, `{"vmoves":[{"id":20,"notification_uuid":"event","instance_uuid":"vm","source_host":"compute-1","dest_host":"compute-3","type":"evacuation","status":"succeeded","start_time":"2026-10-25T12:00:01","end_time":"2026-10-25T12:00:05"},{"id":21,"notification_uuid":"event","instance_uuid":"vm2","type":"evacuation","status":"pending","start_time":null,"end_time":null}]}`)
		}
	}
}

func TestMasakariCollection(t *testing.T) {
	exporter, registry := masakariTestExporter(t, masakariFixtureHandler(t, "", 0), nil, false)
	families := masakariGather(t, registry)
	masakariSample(t, families, "up", nil, 1)
	masakariSample(t, families, "segments", nil, 1)
	masakariSample(t, families, "segment_enabled", nil, 0)
	masakariSample(t, families, "hosts", nil, 2)
	masakariSample(t, families, "reserved_hosts_available", nil, 1)
	masakariSample(t, families, "host_on_maintenance", map[string]string{"uuid": "maintenance"}, 1)
	masakariSample(t, families, "notifications", map[string]string{"type": "VM", "status": "running"}, 1)
	masakariSample(t, families, "notifications", map[string]string{"type": "VM", "status": "failed"}, 0)
	masakariSample(t, families, "vmoves", map[string]string{"type": "evacuation", "status": "succeeded"}, 1)
	masakariSample(t, families, "vmove_duration_seconds", map[string]string{"id": "20"}, 4)
	if len(families["notification"].GetMetric()) != 2 {
		t.Fatal("notifications were not deduplicated")
	}
	if len(families["vmove_duration_seconds"].GetMetric()) != 1 {
		t.Fatal("missing timestamps produced a duration")
	}
	if families["notification_updated_timestamp_seconds"] != nil {
		t.Fatal("null timestamp emitted")
	}
	if exporter.ClientV2.Microversion != "" {
		t.Fatal("shared client microversion mutated")
	}
}

func TestMasakariCollectionFailures(t *testing.T) {
	for _, tc := range []struct {
		path, group string
		code        int
	}{
		{"/v1/segments", "inventory", 403},
		{"/v1/segments/segment/hosts", "inventory", 500},
		{"/v1/notifications", "notifications", 403},
		{"/v1/notifications/event/vmoves", "vmoves", 403},
		{"/v1/notifications/event/vmoves", "vmoves", 404},
		{"/v1/notifications/event/vmoves", "vmoves", 406},
	} {
		t.Run(fmt.Sprintf("%s-%d", tc.group, tc.code), func(t *testing.T) {
			_, registry := masakariTestExporter(t, masakariFixtureHandler(t, tc.path, tc.code), nil, false)
			families := masakariGather(t, registry)
			masakariSample(t, families, "up", nil, 0)
			masakariSample(t, families, "collection_success", map[string]string{"collector": tc.group}, 0)
			if tc.group == "vmoves" && families["vmoves"] != nil {
				t.Fatal("failed collection emitted zero counts")
			}
			if tc.group == "inventory" && families["segments"] != nil {
				t.Fatal("partial inventory emitted")
			}
		})
	}
}

func TestMasakariDisabledMetrics(t *testing.T) {
	_, registry := masakariTestExporter(t, masakariFixtureHandler(t, "", 0), []string{"masakari-host_reserved", "masakari-notification", "masakari-vmove_duration_seconds"}, true)
	families := masakariGather(t, registry)
	masakariSample(t, families, "up", nil, 1)
	for _, name := range []string{"host_reserved", "notification", "vmoves", "vmove", "vmove_duration_seconds"} {
		if families[name] != nil {
			t.Fatalf("disabled metric %s was collected", name)
		}
	}
	if _, err := NewMasakariExporter(&ExporterConfig{}, slog.Default()); err != nil {
		t.Fatal(err)
	}
}

func TestMasakariTimeout(t *testing.T) {
	exporter, registry := masakariTestExporter(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}, nil, false)
	exporter.timeout = 20 * time.Millisecond
	families := masakariGather(t, registry)
	masakariSample(t, families, "up", nil, 0)
}

func TestMasakariNoStaleSamples(t *testing.T) {
	var empty atomic.Bool
	handler := masakariFixtureHandler(t, "", 0)
	_, registry := masakariTestExporter(t, func(w http.ResponseWriter, r *http.Request) {
		if empty.Load() {
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(r.URL.Path, "segments") {
				fmt.Fprint(w, `{"segments":[]}`)
			} else {
				fmt.Fprint(w, `{"notifications":[]}`)
			}
			return
		}
		handler(w, r)
	}, nil, false)
	masakariGather(t, registry)
	empty.Store(true)
	families := masakariGather(t, registry)
	masakariSample(t, families, "segments", nil, 0)
	masakariSample(t, families, "notifications", map[string]string{"type": "COMPUTE_HOST", "status": "finished"}, 0)
	if families["notification"] != nil || families["vmove"] != nil {
		t.Fatal("stale event samples retained")
	}
}

func TestMasakariNotificationHistory(t *testing.T) {
	var mu sync.Mutex
	statusQueries := make(map[string]bool)
	exporter, _ := masakariTestExporter(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		q := r.URL.Query()
		statusQueries[q.Get("status")] = true
		if q.Get("generated-since") != "" {
			t.Error("unexpected history filter with zero lookback")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"notifications":[]}`)
	}, nil, false)
	exporter.lookback = 0
	_, err := exporter.listNotifications(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(statusQueries) != 1 {
		t.Fatalf("unexpected queries %v", statusQueries)
	}
}

func TestMasakariServiceCatalog(t *testing.T) {
	var found bool
	provider := &gophercloud.ProviderClient{EndpointLocator: func(opts gophercloud.EndpointOpts) (string, error) {
		if opts.Type != "instance-ha" || opts.Region != "RegionOne" || opts.Availability != gophercloud.AvailabilityInternal {
			t.Errorf("unexpected endpoint options: %#v", opts)
		}
		found = true
		return "https://masakari.example/v1/", nil
	}}
	opts := gophercloud.EndpointOpts{Region: "RegionOne", Availability: gophercloud.AvailabilityInternal}
	if !IsExporterNameValid("instance-ha") || !isServiceAvailable(provider, opts, "instance-ha") {
		t.Fatal("Masakari not discoverable")
	}
	client, err := openstack.NewInstanceHAV1(provider, opts)
	if err != nil || !found || client.Type != "instance-ha" || client.Endpoint != "https://masakari.example/v1/" {
		t.Fatalf("unexpected service client: %#v, %v", client, err)
	}
}
