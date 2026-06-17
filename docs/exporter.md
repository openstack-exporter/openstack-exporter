# Exporter Internals

<!-- markdownlint-disable MD013 -->

This document describes how exporters are wired, how a scrape is executed, and
how to add a new exporter using the DAG collection framework.

## Runtime Flow

Exporter registration happens during package initialization. Each exporter file
calls `RegisterTypedExporter("<service>", New<Service>Exporter)` from `init()`.
The service name is the OpenStack service/catalog selector such as `compute`,
`network`, or `volume`. Registration also appends that service name to
`SupportedExporters`, which is used by the CLI to create
`--disable-service.<service>` flags and to validate `/probe` include/exclude
filters.

On startup, `main`:

1. Builds service enable/disable flags from `SupportedExporters`.
2. Parses CLI flags and optional Vault password configuration.
3. Resolves the active service list from either Keystone autodetection or
   explicit service flags.
4. Builds one `exporters.ExporterOptions` value with cloud, prefix,
   metric-enable/disable, endpoint, filtering, and per-exporter options.
5. Starts the cache worker if `--cache` is enabled.
6. Starts the HTTP server.

In standalone mode, `buildStandaloneHandler` creates one Prometheus registry,
constructs each enabled exporter once with `exporters.NewExporter`, registers
the exporters and `openstack_exporter_build_info`, and serves that registry for
every `/metrics` request. Keeping the registry alive preserves scrape counters
and histograms across requests.

In multi-cloud mode, `/probe?cloud=<cloud>` selects a cloud per request. The
request can override the configured service list with `include_services` or
`exclude_services`. Without `--cache`, the probe handler caches one Prometheus
handler per `(cloud, service-list)` key in `cloudRegistries`; the first request
for that key constructs exporters and later requests reuse the same registry.
With `--cache`, the probe handler does not scrape OpenStack directly. It reads
the metric families stored by the background cache worker.

In cache mode, `cacheBackgroundService` calls `cache.CollectCache` immediately
and then every half cache TTL. `CollectCache` creates a fresh registry per
service, gathers its metric families, stores them in a new cloud cache, and
atomically swaps that cloud cache into the backend after all enabled services
have been attempted. The HTTP path serializes cached metric families back to
Prometheus text format.

## NewExporter

`exporters.NewExporter(serviceName, opts, logger)` is the common constructor
entry point. It:

1. Loads the selected cloud from `clouds.yaml`.
2. Builds the HTTP transport, including TLS settings and `OS_DEBUG` logging.
3. Creates a Gophercloud service client with `NewServiceClientV2`.
4. Defaults `UUIDGenFunc` when the caller did not provide one.
5. Builds `ExporterConfig`, which embeds `ExporterOptions` and adds the
   resolved service client plus the public service name.
6. Looks up the registered factory by service name and calls it.

There are two names to keep distinct:

* Service name: the public selector registered with `RegisterExporter`, used
  for service flags, catalog lookup, and `/probe` filtering. Examples:
  `compute`, `network`, `volume`.
* Exporter metric namespace: `BaseOpenStackExporter.Name`, used in Prometheus
  metric names and metric enable/disable keys. Examples: `nova`, `neutron`,
  `cinder`.

Metric names are built as:

```text
<prefix>_<exporter-name>_<metric>
```

With the default prefix, Nova's `total_vms` metric becomes
`openstack_nova_total_vms`. The corresponding metric option key is:

```text
<exporter-name>-<metric>
```

For the same metric, that key is `nova-total_vms`. Keep this namespace stable;
users may already have `--disable-metric` and `--enable-metric` values in their
Prometheus jobs.

## Exporter Shape

Most exporters follow this structure:

```go
func init() {
    RegisterTypedExporter("image", NewGlanceExporter)
}

type GlanceExporter struct {
    BaseOpenStackExporter
    sched Schedule
    descs glanceDescs
}

type glanceDescs struct {
    Images         *prometheus.Desc `metric:"images"`
    ImageBytes     *prometheus.Desc `metric:"image_bytes" labels:"id,name,tenant_id" slow:"true"`
    ImageCreatedAt *prometheus.Desc `metric:"image_created_at" labels:"id,name,tenant_id,visibility,hidden,status" slow:"true"`
}

type glanceScrape struct {
    images []images.Image
}
```

The descriptor struct is the metric declaration. `RegisterAndFillDescs` scans
exported `*prometheus.Desc` fields and uses struct tags to create descriptors:

Tag | Meaning
----|--------
`metric` | Local metric name. If omitted, the field name is converted from CamelCase to snake_case.
`labels` | Comma-separated variable label names, in emission order.
`slow:"true"` | Metric is suppressed by `--disable-slow-metrics` unless explicitly re-enabled.
`deprecated:"<version>"` | Metric is suppressed by `--disable-deprecated-metrics` unless explicitly re-enabled.

`RegisterAndFillDescs` also creates the common scrape health descriptors:
`up`, `exporter_scrapes_total`, and `exporter_scrape_errors_total`. When
`--collect-metric-time` is enabled, source fetch durations are emitted through
the shared `openstack_exporter_source_fetch_duration_seconds` metric with
`service` and `source` labels.

Metrics with dynamic label sets can be created manually and added with
`RegisterDesc`. Nova's `server_status` metric does this because
`--nova.metadata-extra-labels` appends labels at runtime.

Descriptor fields are left nil when a metric is disabled, slow-disabled, or
deprecated-disabled. Emitters should call `emitGauge`; it is a no-op for nil
descriptors, which keeps emitter code simple and avoids per-call nil checks.

## DAG Model

The collection framework is declared with three types:

* `Source` fetches OpenStack data into the per-scrape state struct. A source
  may depend on other sources by name with `DependsOn`.
* `Emitter` reads from the per-scrape state and emits Prometheus metrics. It
  declares the metric names it may emit and the source names it needs.
* `Graph` holds the static list of sources and emitters for one exporter.

Example:

```go
var glanceGraph = Graph[*GlanceExporter, glanceScrape]{
    Sources: []Source[*GlanceExporter, glanceScrape]{
        {Name: "images", Fetch: (*GlanceExporter).fetchImages},
    },
    Emitters: []Emitter[*GlanceExporter, glanceScrape]{
        {
            Name:    "count",
            Metrics: []string{"images"},
            Sources: []string{"images"},
            Emit:    (*GlanceExporter).emitCount,
        },
        {
            Name:    "properties",
            Metrics: []string{"image_bytes", "image_created_at"},
            Sources: []string{"images"},
            Emit:    (*GlanceExporter).emitProperties,
        },
    },
}
```

Constructor flow:

1. Build the concrete exporter and embedded `BaseOpenStackExporter`.
2. Call `RegisterAndFillDescs(&e.descs)`.
3. Call `graph.PruneSchedule(&e.BaseOpenStackExporter)`.
4. Store the pruned schedule on the exporter.
5. Call `graph.LogDAG(...)` for debug visibility.

Scrape flow:

1. `Collect` allocates a fresh scrape state struct.
2. `RunCollect` wraps collection with scrape counters, duration, error count,
   and `up` emission.
3. `runSchedule` starts every source or emitter whose dependencies are
   satisfied.
4. Each source writes fetched data into the scrape state.
5. Each emitter reads the source data it declared and emits metrics as soon as
   those sources complete.

The scheduler validates the combined source/emitter DAG with Kahn's algorithm.
It stores topological waves for debug logging, but execution is
dependency-driven rather than wave-barrier-driven: a node starts as soon as its
own predecessors have finished, even when unrelated earlier-wave work is still
running.

`PruneSchedule` makes metric options affect runtime work:

1. An emitter is live when at least one of its declared metrics is enabled.
2. A source is live when a live emitter depends on it, directly or through
   source dependencies.
3. Disabled emitters and unused sources are removed from the stored schedule.

This means disabling all metrics backed by an expensive source removes the
source fetch as well. For extra-expensive API calls inside a shared source, keep
an explicit `IsMetricEnabled(...)` guard in the fetch method too.

Source and emitter conventions:

* Sources should write to disjoint fields of the scrape state, so concurrent
  sources stay race-free.
* If a source reads data populated by another source, it must declare that
  source in `DependsOn`.
* Emitters should treat the scrape state as read-only.
* Emitters should not perform OpenStack API requests; make those requests in a
  source and have the emitter read the scrape state.
* Emitters should emit through `emitGauge` unless they intentionally need a
  different Prometheus value type.
* A source returns an error when its API fetch fails. `RunCollect` reports
  `up=0` when no sources are scheduled or all scheduled sources fail.
* Emitter errors are logged but do not currently affect the source failure
  count.

## Adding a New Exporter

1. Create `exporters/<name>.go`.
2. Register the OpenStack service name in `init()`:

   ```go
   func init() {
       RegisterTypedExporter("my-service", NewMyExporter)
   }
   ```

3. Define the exporter struct with `BaseOpenStackExporter`, a `Schedule`,
   and a descriptor struct.
4. Define the descriptor struct with `metric`, `labels`, `slow`, and
   `deprecated` tags as needed.
5. Define a per-scrape state struct. Keep API response slices and derived data
   there, not on the exporter.
6. Declare a `Graph` with source nodes and emitter nodes.
7. In the constructor:

   ```go
   func NewMyExporter(config *ExporterConfig, logger *slog.Logger) (*MyExporter, error) {
       e := &MyExporter{
           BaseOpenStackExporter: BaseOpenStackExporter{
               Name:           "my_metric_namespace",
               ExporterConfig: *config,
               logger:         logger,
           },
       }
       e.RegisterAndFillDescs(&e.descs)
       sched, err := myGraph.PruneSchedule(&e.BaseOpenStackExporter)
       if err != nil {
           return nil, err
       }
       e.sched = sched
       myGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)
       return e, nil
   }
   ```

8. Implement `Collect`:

   ```go
   func (e *MyExporter) Collect(ch chan<- prometheus.Metric) {
       e.BaseOpenStackExporter.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
           s := new(myScrape)
           return runSchedule(e, &e.BaseOpenStackExporter, &myGraph, e.sched, s, ch)
       })
   }
   ```

9. Implement source methods that fetch OpenStack data into `*myScrape`.
10. Implement emitter methods that read `*myScrape` and emit metrics.
11. Add fixtures and a test suite under `exporters/`.
12. Add the suite to `TestOpenStackSuites` in `exporters/exporter_test.go`.
13. Regenerate metrics documentation:

    ```sh
    go run ./script/generate-metrics-doc.go
    ```

14. Run focused tests:

    ```sh
    env -u OS_COMPUTE_API_VERSION go test . ./cache ./exporters ./utils
    ```

## Naming Checklist

Before opening a PR, check these names:

* `RegisterTypedExporter("<service>", ...)` uses the OpenStack service/catalog selector.
* `BaseOpenStackExporter.Name` uses the stable Prometheus metric namespace.
* Metric option keys use `BaseOpenStackExporter.Name`, not the service name.
* Descriptor `metric` tags use the local metric name without prefix or exporter.
* Emitter `Metrics` entries exactly match descriptor local metric names.
* Test expected output uses full Prometheus names:
  `openstack_<exporter-name>_<metric>`.

The generated metrics documentation lists both full metric names and option keys
in [metrics.md](metrics.md).
