package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"log/slog"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/openstack-exporter/openstack-exporter/cache"
	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
	"golang.org/x/sync/errgroup"
)

var defaultEnabledServices = []string{"network", "compute", "image", "volume", "identity", "object-store", "load-balancer", "container-infra", "dns", "baremetal", "gnocchi", "database", "orchestration", "placement", "sharev2"}

var DEFAULT_OS_CLIENT_CONFIG = "/etc/openstack/clouds.yaml"

var (
	metrics                  = kingpin.Flag("web.telemetry-path", "uri path to expose metrics").Default("/metrics").String()
	osClientConfig           = kingpin.Flag("os-client-config", "Path to the cloud configuration file").Default(DEFAULT_OS_CLIENT_CONFIG).String()
	prefix                   = kingpin.Flag("prefix", "Prefix for metrics").Default("openstack").String()
	endpointType             = kingpin.Flag("endpoint-type", "openstack endpoint type to use (i.e: public, internal, admin)").Default("public").String()
	collectTime              = kingpin.Flag("collect-metric-time", "time spent collecting each metric").Default("false").Bool()
	disabledMetrics          = kingpin.Flag("disable-metric", "multiple --disable-metric can be specified in the format: service-metric (i.e: cinder-snapshots)").Default("").Short('d').Strings()
	disableSlowMetrics       = kingpin.Flag("disable-slow-metrics", "Disable slow metrics for performance reasons").Default("false").Bool()
	disableDeprecatedMetrics = kingpin.Flag("disable-deprecated-metrics", "Disable deprecated metrics").Default("false").Bool()
	disableCinderAgentUUID   = kingpin.Flag("disable-cinder-agent-uuid", "Disable UUID generation for Cinder agents").Default("false").Bool()
	cloud                    = kingpin.Arg("cloud", "name or id of the cloud to gather metrics from").String()
	multiCloud               = kingpin.Flag("multi-cloud", "Toggle the multiple cloud scraping mode under /probe?cloud=").Default("false").Bool()
	domainID                 = kingpin.Flag("domain-id", "Gather metrics only for the given Domain ID (defaults to all domains)").String()
	cacheEnable              = kingpin.Flag("cache", "Enable Cache mechanism globally").Default("false").Bool()
	cacheTTL                 = kingpin.Flag("cache-ttl", "TTL duration for cache expiry(eg. 10s, 11m, 1h)").Default("300s").Duration()
	tenantID                 = kingpin.Flag("tenant-id", "Gather metrics only for the given Tenant ID (default to all tenants)").String()
	novaMetadataMapping      = utils.LabelMapping(kingpin.Flag("nova.metadata-extra-labels", "Map provided server metadata keys to labels in openstack_nova_server_status metric").PlaceHolder("LABEL=KEY,KEY").Default(""))
)

func main() {
	services := make(map[string]*bool)

	for _, service := range defaultEnabledServices {
		flagName := fmt.Sprintf("disable-service.%s", service)
		flagHelp := fmt.Sprintf("Disable the %s service exporter", service)
		services[service] = kingpin.Flag(flagName, flagHelp).Default().Bool()
	}
	toolkitFlags := webflag.AddFlags(kingpin.CommandLine, ":9180")

	promlogConfig := &promslog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("openstack-exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promslog.New(promlogConfig)
	logger.Info("Build Version", "version_info", version.Info(), "build_context", version.BuildContext())

	logger.Info("Starting openstack exporter version for cloud", "version", version.Info(), "cloud", *cloud)
	logger.Info("Build context", "build_context", version.BuildContext())

	if *cloud == "" && !*multiCloud {
		logger.Error("openstack-exporter: error: required argument 'cloud' or flag --multi-cloud not provided, try --help")
		os.Exit(1)
	}

	if *osClientConfig != DEFAULT_OS_CLIENT_CONFIG {
		logger.Debug("Setting Env var OS_CLIENT_CONFIG_FILE", "os_client_config_file", *osClientConfig)
		os.Setenv("OS_CLIENT_CONFIG_FILE", *osClientConfig)
	}

	if _, err := os.Stat(*osClientConfig); err != nil {
		logger.Error("Could not read config file", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)

	// Start the backend service.
	if *cacheEnable {
		go cacheBackgroundService(ctx, services, errChan, logger)
	}

	var registryMap map[string]*prometheus.Registry
	commonRegistry := prometheus.NewPedanticRegistry()
	commonExporter := exporters.NewCommonMetricsExporter(*prefix)
	commonRegistry.MustRegister(
		commonExporter,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	var err error
	if *multiCloud {
		registryMap, err = initMultiCloudRegistries(
			services,
			*prefix, *disabledMetrics,
			*endpointType,
			*collectTime, *disableSlowMetrics, *disableDeprecatedMetrics, *disableCinderAgentUUID,
			*domainID, *tenantID,
			logger,
		)
		if err != nil {
			logger.Error("failed to initialize multiple cloud registries", "error", err.Error())
			os.Exit(1)
		}
	} else {
		commonRegistry, err = initSingleCloudRegistry(
			services,
			*prefix, *cloud,
			*disabledMetrics,
			*endpointType,
			*collectTime, *disableSlowMetrics, *disableDeprecatedMetrics, *disableCinderAgentUUID,
			*domainID, *tenantID,
			logger,
			commonExporter,
		)
		if err != nil {
			logger.Error("failed to initialize single cloud registries", "error", err.Error())
			os.Exit(1)
		}
	}

	// Start the HTTP server.
	go startHTTPServer(ctx, services, commonRegistry, registryMap, commonExporter, toolkitFlags, errChan, logger)

	// Wait for an error from any service or a termination signal.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.Error("Shutting down due to error", "err", err)
		cancel()
	case <-sigChan:
		logger.Info("Termination signal received. Shutting down...")
		cancel()
	}
}

// cacheBackgroundService runs a background service to collect the metrics and stores in the cache.
// It collects data every cache-ttl/2 time and flush every cache-ttl time.
// The cache data will be read by the Prometheus HandleFunc.
func cacheBackgroundService(ctx context.Context, services map[string]*bool, errChan chan<- error, logger *slog.Logger) {
	logger.Info("Start cache background service")
	collectTicker := time.NewTicker(*cacheTTL / 2)
	defer collectTicker.Stop()
	ttlTicker := time.NewTicker(*cacheTTL)
	defer ttlTicker.Stop()

	// Collect cache data in the beginning.
	if err := cache.CollectCache(exporters.EnableExporter, *multiCloud, services, *prefix, *cloud, *disabledMetrics, *endpointType, *collectTime, *disableSlowMetrics, *disableDeprecatedMetrics, *disableCinderAgentUUID, *domainID, *tenantID, novaMetadataMapping, nil, logger); err != nil {
		logger.Error("Failed to collect from cache", "err", err)
		errChan <- err
		return
	}

	for {
		select {
		case <-collectTicker.C:
			if err := cache.CollectCache(exporters.EnableExporter, *multiCloud, services, *prefix, *cloud, *disabledMetrics, *endpointType, *collectTime, *disableSlowMetrics, *disableDeprecatedMetrics, *disableCinderAgentUUID, *domainID, *tenantID, novaMetadataMapping, nil, logger); err != nil {
				errChan <- err
				return
			}
		case <-ttlTicker.C:
			cache.FlushExpiredCloudCaches(*cacheTTL)
			logger.Info("Cache TTL flush")
		case <-ctx.Done():
			logger.Info("Backend service is stopping")
			return
		}
	}
}

func startHTTPServer(
	ctx context.Context,
	services map[string]*bool,
	commonRegistry *prometheus.Registry,
	registryMap map[string]*prometheus.Registry,
	commonExporter *exporters.CommonMetricsExporter,
	toolkitFlags *web.FlagConfig,
	errChan chan<- error,
	logger *slog.Logger,
) {
	links := []web.LandingLinks{}

	if *multiCloud {
		http.HandleFunc("/probe", probeHandler(services, registryMap, commonExporter, logger))
		http.Handle(*metrics, promhttp.HandlerFor(commonRegistry, promhttp.HandlerOpts{}))
		logger.Info("openstack exporter started in multi cloud mode (/probe?cloud=)")
		links = append(links, web.LandingLinks{
			Address: *metrics,
			Text:    "Metrics",
		}, web.LandingLinks{
			Address: "/probe",
			Text:    "Probes",
		})
	} else {
		http.HandleFunc(*metrics, metricHandler(services, commonRegistry, commonExporter, logger))
		logger.Info("openstack exporter started in legacy mode")
		links = append(links, web.LandingLinks{
			Address: *metrics,
			Text:    "Metrics",
		})
	}

	if *metrics != "/" && *metrics != "" {
		landingConfig := web.LandingConfig{
			Name:        "openstack_exporter",
			Description: "Prometheus Exporter for openstack",
			Version:     version.Info(),
			Links:       links,
		}

		landingPage, err := web.NewLandingPage(landingConfig)
		if err != nil {
			logger.Error("Failed to create landing page", "error", err)
			os.Exit(1)
		}
		http.Handle("/", landingPage)
	}

	if *domainID != "" {
		logger.Info("Gathering metrics for configured domain ID", "domain_id", *domainID)
	}

	if *tenantID != "" {
		logger.Info("Gathering metrics for configured tenant ID", "tenant_id", *tenantID)
	}

	srv := &http.Server{}
	go func() {
		if err := web.ListenAndServe(srv, toolkitFlags, logger); err != nil {
			if ctx.Err() != nil {
				return
			}
			errChan <- err
		}
	}()
	<-ctx.Done()
	if err := srv.Shutdown(context.Background()); err != nil {
		logger.Error("HTTP server shutdown error", "err", err)
	}
}

func probeHandler(
	services map[string]*bool,
	registryMap map[string]*prometheus.Registry,
	commonExporter *exporters.CommonMetricsExporter,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer observeScrape(commonExporter)()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		r = r.WithContext(ctx)

		cloud := r.URL.Query().Get("cloud")
		if cloud == "" {
			http.Error(w, "'cloud' parameter is missing", http.StatusBadRequest)
			return
		}

		enabledServices := getEnabledServices(services)

		if include := r.URL.Query().Get("include_services"); include != "" {
			enabledServices = strings.Split(include, ",")
		}
		excludes := strings.Split(r.URL.Query().Get("exclude_services"), ",")
		enabledServices = exporters.RemoveElements(enabledServices, excludes)

		logger.Info("Filtered services", "enabled_services", strings.Join(enabledServices, ","))

		excludeServices := strings.Split(r.URL.Query().Get("exclude_services"), ",")
		enabledServices = exporters.RemoveElements(enabledServices, excludeServices)
		logger.Info("Enabled services", "enabled_services", enabledServices)

		// Get data from cache
		if *cacheEnable {
			logger.Info("Serving from cache", "cloud", cloud)
			if err := cache.WriteCacheToResponse(w, r, cloud, enabledServices, logger); err != nil {
				commonExporter.ScrapeErrors().Inc()
				http.Error(w, "Failed to serve from cache", http.StatusInternalServerError)
				logger.Error("Write cache to response failed", "error", err)
			}
			return
		}

		defer withPanicRecovery(w, commonExporter, logger)

		reg, ok := registryMap[cloud]
		if !ok {
			http.Error(w, "Unknown cloud: "+cloud, http.StatusNotFound)
			commonExporter.ScrapeErrors().Inc()
			return
		}
		promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	}
}

func metricHandler(services map[string]*bool, registry *prometheus.Registry, commonExporter *exporters.CommonMetricsExporter, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer observeScrape(commonExporter)()

		enabledServices := getEnabledServices(services)

		if *osClientConfig != DEFAULT_OS_CLIENT_CONFIG {
			logger.Debug("Setting Env var OS_CLIENT_CONFIG_FILE", "os_client_config_file", *osClientConfig)
			os.Setenv("OS_CLIENT_CONFIG_FILE", *osClientConfig)
		}

		if *cacheEnable {
			if err := cache.WriteCacheToResponse(w, r, *cloud, enabledServices, logger); err != nil {
				commonExporter.ScrapeErrors().Inc()
				logger.Error("Write cache to response failed", "error", err)
			}
			return
		}

		defer withPanicRecovery(w, commonExporter, logger)

		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	}
}

func buildAndValidateExporters(
	enabledServices []string,
	prefix, cloud string,
	disabledMetrics []string,
	endpointType string,
	collectTime, disableSlowMetrics, disableDeprecatedMetrics, disableCinderAgentUUID bool,
	domainID, tenantID string,
	logger *slog.Logger,
) (*prometheus.Registry, int) {
	registry := prometheus.NewPedanticRegistry()

	var (
		mu      sync.Mutex
		enabled int32
		g       errgroup.Group
	)

	for _, enabledService := range enabledServices {
		svc := enabledService
		g.Go(func() error {
			exp, err := exporters.EnableExporter(
				svc, prefix, cloud,
				disabledMetrics, endpointType,
				collectTime, disableSlowMetrics,
				disableDeprecatedMetrics, disableCinderAgentUUID,
				domainID, tenantID, novaMetadataMapping,
				nil,
				logger,
			)
			if err != nil {
				logger.Error("enabling exporter for service failed", "cloud", cloud, "service", svc, "err", err)
				return nil
			}

			mu.Lock()
			registry.MustRegister(*exp)
			mu.Unlock()

			atomic.AddInt32(&enabled, 1)
			logger.Info("Enabled exporter for service", "cloud", cloud, "service", svc)
			return nil
		})
	}

	_ = g.Wait()
	return registry, int(enabled)
}

func getEnabledServices(services map[string]*bool) []string {
	var result []string
	for s, disabled := range services {
		if !*disabled {
			result = append(result, s)
		}
	}
	return result
}

func withPanicRecovery(w http.ResponseWriter, exporter *exporters.CommonMetricsExporter, logger *slog.Logger) {
	if rec := recover(); rec != nil {
		exporter.ScrapeErrors().Inc()
		logger.Error("Recovered from panic", "recover", rec)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func observeScrape(exporter *exporters.CommonMetricsExporter) func() {
	start := time.Now()
	exporter.TotalScrapes().Inc()
	return func() {
		durationMs := float64(time.Since(start).Milliseconds())
		exporter.ScrapeDuration().Observe(durationMs)
	}
}

func registerCommonMetrics(reg *prometheus.Registry, commonExporter *exporters.CommonMetricsExporter, logger *slog.Logger) {
	reg.MustRegister(
		commonExporter,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	metrics, err := reg.Gather()
	if err != nil {
		logger.Error("Failed to gather metrics from registry", "error", err)
		os.Exit(1)
	}
	if len(metrics) == 0 {
		logger.Error("Registry is empty after initializing exporters, exiting")
		os.Exit(1)
	}
}

func initSingleCloudRegistry(
	services map[string]*bool,
	prefix, cloud string,
	disabledMetrics []string,
	endpointType string,
	collectTime, disableSlowMetrics, disableDeprecatedMetrics, disableCinderAgentUUID bool,
	domainID, tenantID string,
	logger *slog.Logger,
	commonExporter *exporters.CommonMetricsExporter,
) (*prometheus.Registry, error) {
	enabledServices := getEnabledServices(services)

	registry, enabled := buildAndValidateExporters(
		enabledServices,
		prefix, cloud,
		disabledMetrics,
		endpointType,
		collectTime, disableSlowMetrics, disableDeprecatedMetrics, disableCinderAgentUUID,
		domainID, tenantID,
		logger,
	)
	if enabled == 0 {
		return nil, fmt.Errorf("no exporters enabled for cloud: %s", cloud)
	}

	registerCommonMetrics(registry, commonExporter, logger)
	return registry, nil
}

func initMultiCloudRegistries(
	services map[string]*bool,
	prefix string,
	disabledMetrics []string,
	endpointType string,
	collectTime, disableSlowMetrics, disableDeprecatedMetrics, disableCinderAgentUUID bool,
	domainID, tenantID string,
	logger *slog.Logger,
) (map[string]*prometheus.Registry, error) {
	allClouds, err := clientconfig.LoadCloudsYAML()
	if err != nil {
		return nil, fmt.Errorf("failed to parse os-client-config file: %w", err)
	}

	registryMap := make(map[string]*prometheus.Registry)
	var mu sync.Mutex
	var g errgroup.Group

	for name := range allClouds {
		cloudName := name
		g.Go(func() error {
			enabledServices := getEnabledServices(services)

			reg, enabled := buildAndValidateExporters(
				enabledServices,
				prefix, cloudName,
				disabledMetrics,
				endpointType,
				collectTime, disableSlowMetrics, disableDeprecatedMetrics, disableCinderAgentUUID,
				domainID, tenantID,
				logger,
			)

			if enabled == 0 {
				logger.Warn("no exporters enabled for cloud", "cloud", cloudName)
				return nil
			}

			mu.Lock()
			registryMap[cloudName] = reg
			mu.Unlock()

			logger.Info("initialized exporters for cloud", "cloud", cloudName)
			return nil
		})
	}

	_ = g.Wait()

	if len(registryMap) == 0 {
		return nil, fmt.Errorf("no exporters initialized for any cloud")
	}
	return registryMap, nil
}
