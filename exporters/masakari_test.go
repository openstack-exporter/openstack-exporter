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
	"github.com/gophercloud/gophercloud/v2/openstack/instanceha/v1/segments"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

const masakariTestNotification = `{"id":11,"notification_uuid":"event","source_host_uuid":"host","type":"COMPUTE_HOST","status":"finished","generated_time":"2026-10-25T12:00:00.123456","updated_at":null}`
const masakariOldNotification = `{"id":12,"notification_uuid":"old","source_host_uuid":"host","type":"VM","status":"running","generated_time":"2020-01-01T00:00:00","updated_at":null}`

func masakariTestExporter(t *testing.T, handler http.HandlerFunc, disabled []string, slow bool) (*MasakariExporter, *prometheus.Registry) {
	t.Helper()
	return masakariTestExporterWithVersion(t, handler, disabled, slow, "1.3")
}

func masakariTestExporterWithVersion(t *testing.T, handler http.HandlerFunc, disabled []string, slow bool, maxVersion string) (*MasakariExporter, *prometheus.Registry) {
	t.Helper()
	previousLookback, previousTimeout, previousVMoves := *masakariLookback, *masakariTimeout, *masakariVMoves
	*masakariLookback, *masakariTimeout, *masakariVMoves = 24*time.Hour, 30*time.Second, true
	t.Cleanup(func() {
		*masakariLookback, *masakariTimeout, *masakariVMoves = previousLookback, previousTimeout, previousVMoves
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"versions":[{"id":"v1.0","status":"CURRENT","min_version":"1.0","version":%q}]}`, maxVersion)
			return
		}
		handler(w, r)
	}))
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
	if exporter.ClientV2.Microversion != "1.3" {
		t.Fatal("unexpected negotiated microversion")
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
			masakariSample(t, families, "up", nil, 1)
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

func TestMasakariProjectScopedDiscovery(t *testing.T) {
	previousLookback, previousTimeout := *masakariLookback, *masakariTimeout
	*masakariLookback, *masakariTimeout = 24*time.Hour, 30*time.Second
	t.Cleanup(func() {
		*masakariLookback, *masakariTimeout = previousLookback, previousTimeout
	})
	for _, prefix := range []string{"", "/masakari"} {
		t.Run(prefix, func(t *testing.T) {
			var discovered, listed atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case prefix + "/v1/":
					discovered.Store(true)
					fmt.Fprint(w, `{"version":{"id":"v1.0","status":"CURRENT","min_version":"1.0","version":"1.3"}}`)
				case prefix + "/v1/project/segments":
					listed.Store(true)
					if r.Header.Get("OpenStack-API-Version") != "instance-ha 1.3" {
						t.Error("resource request did not use negotiated microversion")
					}
					fmt.Fprint(w, `{"segments":[]}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{}, Endpoint: server.URL + prefix + "/v1/project/", Type: "instance-ha"}
			exporter, err := NewMasakariExporter(&ExporterConfig{ClientV2: client, Prefix: "openstack"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			if err != nil {
				t.Fatal(err)
			}
			if exporter.ClientV2.Endpoint != client.Endpoint || client.Microversion != "" {
				t.Fatal("discovery changed the resource endpoint or caller's client")
			}
			if _, err := segments.List(exporter.ClientV2, nil).AllPages(context.Background()); err != nil {
				t.Fatal(err)
			}
			if !discovered.Load() || !listed.Load() {
				t.Fatal("expected version-root discovery and project-scoped resource listing")
			}
		})
	}
}

func TestMasakariMicroversions(t *testing.T) {
	for _, version := range []string{"1.0", "1.1", "1.2", "1.3"} {
		t.Run(version, func(t *testing.T) {
			handler := masakariFixtureHandler(t, "", 0)
			exporter, registry := masakariTestExporterWithVersion(t, func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("OpenStack-API-Version"); got != "instance-ha "+version {
					t.Errorf("API version header = %q, want instance-ha %s", got, version)
				}
				if version < "1.3" && strings.HasSuffix(r.URL.Path, "/vmoves") {
					t.Error("VM moves requested on an unsupported API")
				}
				handler(w, r)
			}, nil, false, version)
			families := masakariGather(t, registry)
			masakariSample(t, families, "up", nil, 1)
			if _, ok := exporter.Metrics["segment_enabled"]; ok != (version >= "1.2") {
				t.Fatal("segment_enabled does not respect API 1.2")
			}
			if _, ok := exporter.Metrics["vmoves"]; ok != (version >= "1.3") {
				t.Fatal("VM moves do not respect API 1.3")
			}
		})
	}
}

func TestMasakariMicroversionOverride(t *testing.T) {
	t.Setenv("OS_INSTANCE_HA_API_VERSION", "1.2")
	exporter, registry := masakariTestExporter(t, masakariFixtureHandler(t, "", 0), nil, false)
	masakariSample(t, masakariGather(t, registry), "up", nil, 1)
	if exporter.collectVMoves || exporter.ClientV2.Microversion != "1.2" {
		t.Fatal("explicit microversion was not respected")
	}
	config := exporter.ExporterConfig
	t.Setenv("OS_INSTANCE_HA_API_VERSION", "1.4")
	if _, err := NewMasakariExporter(&config, slog.Default()); err == nil {
		t.Fatal("unsupported explicit microversion accepted")
	}
	if config.ClientV2.Microversion != "1.2" {
		t.Fatal("constructor mutated the caller's client")
	}
}

func TestMasakariPagination(t *testing.T) {
	pages := make(map[string]int)
	_, registry := masakariTestExporter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("status") != "" {
			fmt.Fprint(w, `{"notifications":[]}`)
			return
		}
		pages[r.URL.Path]++
		marker := r.URL.Query().Get("marker")
		switch marker {
		case "", "1":
			id, uuid := 1, "first"
			if marker == "1" {
				id, uuid = 2, "second"
			}
			switch r.URL.Path {
			case "/v1/segments":
				fmt.Fprintf(w, `{"segments":[{"id":%d,"uuid":%q}]}`, id, uuid)
			case "/v1/segments/first/hosts", "/v1/segments/second/hosts":
				fmt.Fprintf(w, `{"hosts":[{"id":%d,"uuid":%q}]}`, id, uuid)
			case "/v1/notifications":
				fmt.Fprintf(w, `{"notifications":[{"id":%d,"notification_uuid":%q,"type":"COMPUTE_HOST","status":"finished"}]}`, id, uuid)
			case "/v1/notifications/first/vmoves", "/v1/notifications/second/vmoves":
				fmt.Fprintf(w, `{"vmoves":[{"id":%d,"notification_uuid":%q,"type":"evacuation","status":"succeeded"}]}`, id, strings.Split(r.URL.Path, "/")[3])
			default:
				t.Errorf("unexpected path %s", r.URL.Path)
			}
		case "2":
			key := "hosts"
			switch r.URL.Path {
			case "/v1/segments":
				key = "segments"
			case "/v1/notifications":
				key = "notifications"
			default:
				if strings.HasSuffix(r.URL.Path, "/vmoves") {
					key = "vmoves"
				}
			}
			fmt.Fprintf(w, `{"%s":[]}`, key)
		default:
			t.Errorf("unexpected marker %q", marker)
		}
	}, nil, false)
	families := masakariGather(t, registry)
	masakariSample(t, families, "segments", nil, 2)
	masakariSample(t, families, "hosts", map[string]string{"segment_uuid": "second"}, 2)
	masakariSample(t, families, "notifications", map[string]string{"type": "COMPUTE_HOST", "status": "finished"}, 2)
	masakariSample(t, families, "vmoves", map[string]string{"type": "evacuation", "status": "succeeded"}, 4)
	if len(pages) != 6 {
		t.Fatalf("expected all resource paths, got %v", pages)
	}
	for path, count := range pages {
		if count != 3 {
			t.Errorf("%s requested %d pages, want 3", path, count)
		}
	}
}

func TestMasakariDisabledCollectors(t *testing.T) {
	disabled := make([]string, 0, len(defaultMasakariMetrics))
	for _, metric := range defaultMasakariMetrics {
		disabled = append(disabled, "masakari-"+metric.Name)
	}
	_, registry := masakariTestExporter(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("disabled collector made a request: %s", r.URL)
	}, disabled, false)
	masakariSample(t, masakariGather(t, registry), "up", nil, 0)
}

func TestMasakariCollectionTiming(t *testing.T) {
	exporter, registry := masakariTestExporter(t, masakariFixtureHandler(t, "", 0), nil, false)
	exporter.CollectTime = true
	families := masakariGather(t, registry)
	if got := len(families["openstack_metric_collect_seconds"].GetMetric()); got != 3 {
		t.Fatalf("got %d collector timings, want 3", got)
	}
}
