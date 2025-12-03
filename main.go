package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"log/slog"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/openstack-exporter/openstack-exporter/cache"
	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
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
	dnsConcurrentCount       = kingpin.Flag("dns-concurrent-count", "concurrent count for dns metric").Default("10").Int()
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

	if *cloud == "" && !*multiCloud {
		logger.Error("openstack-exporter: error: required argument 'cloud' or flag --multi-cloud not provided, try --help")
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

	// Start the HTTP server.
	go startHTTPServer(ctx, services, toolkitFlags, errChan, logger)

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
	if err := cache.CollectCache(exporters.EnableExporter, *multiCloud, services, *prefix, *cloud, *disabledMetrics, *endpointType, *collectTime, *disableSlowMetrics, *disableDeprecatedMetrics, *disableCinderAgentUUID, *domainID, *tenantID, novaMetadataMapping, *dnsConcurrentCount, nil, logger); err != nil {
		logger.Error("Failed to collect from cache", "err", err)
		errChan <- err
		return
	}

	for {
		select {
		case <-collectTicker.C:
			if err := cache.CollectCache(exporters.EnableExporter, *multiCloud, services, *prefix, *cloud, *disabledMetrics, *endpointType, *collectTime, *disableSlowMetrics, *disableDeprecatedMetrics, *disableCinderAgentUUID, *domainID, *tenantID, novaMetadataMapping, *dnsConcurrentCount, nil, logger); err != nil {
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

func startHTTPServer(ctx context.Context, services map[string]*bool, toolkitFlags *web.FlagConfig, errChan chan<- error, logger *slog.Logger) {
	links := []web.LandingLinks{}

	if *multiCloud {
		http.HandleFunc("/probe", probeHandler(services, logger))
		http.Handle(*metrics, promhttp.Handler())
		logger.Info("openstack exporter started in multi cloud mode (/probe?cloud=)")
		links = append(links, web.LandingLinks{
			Address: *metrics,
			Text:    "Metrics",
		}, web.LandingLinks{
			Address: "/probe",
			Text:    "Probes",
		})
	} else {
		logger.Info("openstack exporter started in legacy mode")
		http.HandleFunc(*metrics, metricHandler(services, logger))
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
			logger.Error("Failed to start webserver", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	if err := srv.Shutdown(context.Background()); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}
}

func probeHandler(services map[string]*bool, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		r = r.WithContext(ctx)

		cloud := r.URL.Query().Get("cloud")
		if cloud == "" {
			http.Error(w, "'cloud' parameter is missing", http.StatusBadRequest)
			return
		}

		enabledServices := []string{}

		for service, disabled := range services {
			if !*disabled {
				enabledServices = append(enabledServices, service)
			}
		}

		includeServices := r.URL.Query().Get("include_services")
		if includeServices != "" {
			enabledServices = strings.Split(includeServices, ",")
		}

		excludeServices := strings.Split(r.URL.Query().Get("exclude_services"), ",")
		enabledServices = exporters.RemoveElements(enabledServices, excludeServices)
		logger.Info("Enabled services", "enabled_services", enabledServices)

		// Get data from cache
		if *cacheEnable {
			if err := cache.WriteCacheToResponse(w, r, cloud, enabledServices, logger); err != nil {
				logger.Error("Write cache to response failed", "error", err)
			}
			return
		}

		registry := prometheus.NewPedanticRegistry()
		for _, service := range enabledServices {
			exp, err := exporters.EnableExporter(service, *prefix, cloud, *disabledMetrics, *endpointType, *collectTime, *disableSlowMetrics, *disableDeprecatedMetrics, *disableCinderAgentUUID, *domainID, *tenantID, novaMetadataMapping, *dnsConcurrentCount, nil, logger)
			if err != nil {
				logger.Error("Enabling exporter for service failed", "service", service, "error", err)
				continue
			}
			registry.MustRegister(*exp)
			logger.Info("Enabled exporter for service", "service", service)
		}

		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}

func metricHandler(services map[string]*bool, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Starting openstack exporter version for cloud", "version", version.Info(), "cloud", *cloud)
		logger.Info("Build context", "build_context", version.BuildContext())

		if *osClientConfig != DEFAULT_OS_CLIENT_CONFIG {
			logger.Debug("Setting Env var OS_CLIENT_CONFIG_FILE", "os_client_config_file", *osClientConfig)
			os.Setenv("OS_CLIENT_CONFIG_FILE", *osClientConfig)
		}

		enabledServices := []string{}
		for service, disabled := range services {
			if !*disabled {
				enabledServices = append(enabledServices, service)
			}
		}

		// Get data from cache
		if *cacheEnable {
			if err := cache.WriteCacheToResponse(w, r, *cloud, enabledServices, logger); err != nil {
				logger.Error("Write cache to response failed", "error", err)
			}
			return
		}

		registry := prometheus.NewPedanticRegistry()
		enabledExporters := 0
		for _, service := range enabledServices {
			exp, err := exporters.EnableExporter(service, *prefix, *cloud, *disabledMetrics, *endpointType, *collectTime, *disableSlowMetrics, *disableDeprecatedMetrics, *disableCinderAgentUUID, *domainID, *tenantID, novaMetadataMapping, *dnsConcurrentCount, nil, logger)
			if err != nil {
				// Log error and continue with enabling other exporters
				logger.Error("enabling exporter for service failed", "service", service, "error", err)
				continue
			}
			registry.MustRegister(*exp)
			logger.Info("Enabled exporter for service", "service", service)
			enabledExporters++
		}

		if enabledExporters == 0 {
			logger.Error("No exporter has been enabled, exiting")
			os.Exit(-1)
		}

		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}
