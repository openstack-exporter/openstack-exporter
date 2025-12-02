package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/openstack-exporter/openstack-exporter/cache"
	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
	pver "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

const (
	progName                 = "openstack-exporter"
	DEFAULT_OS_CLIENT_CONFIG = "/etc/openstack/clouds.yaml"
)

var defaultEnabledServices = []string{"network", "compute", "image", "volume", "identity", "object-store", "load-balancer", "container-infra", "dns", "baremetal", "gnocchi", "database", "orchestration", "placement", "sharev2"}

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
	kingpin.Version(version.Print(progName))
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

	ctx1, cancel1 := context.WithCancelCause(context.Background())
	defer cancel1(nil)

	ctx2, cancel2 := signal.NotifyContext(ctx1, syscall.SIGINT, syscall.SIGTERM)
	defer cancel2()

	// Start the backend service.
	if *cacheEnable {
		go cacheBackgroundService(ctx2, services, cancel1, logger)
	}

	// Start the HTTP server.
	go startHTTPServer(ctx2, services, toolkitFlags, cancel1, logger)

	<-ctx2.Done()
	if err := context.Cause(ctx2); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("Shutting down due to error", "error", err)
		os.Exit(1)
	} else {
		logger.Info("Termination signal received. Shutting down...")
	}
}

// cacheBackgroundService runs a background service to collect the metrics and stores in the cache.
// It collects data every cache-ttl/2 time and flush every cache-ttl time.
// The cache data will be read by the Prometheus HandleFunc.
func cacheBackgroundService(ctx context.Context, services map[string]*bool, cancel context.CancelCauseFunc, logger *slog.Logger) {
	logger.Info("Start cache background service")
	collectTicker := time.NewTicker(*cacheTTL / 2)
	defer collectTicker.Stop()
	ttlTicker := time.NewTicker(*cacheTTL)
	defer ttlTicker.Stop()

	// Collect cache data in the beginning.
	if err := cache.CollectCache(exporters.EnableExporter, *multiCloud, services, *prefix, *cloud, *disabledMetrics, *endpointType, *collectTime, *disableSlowMetrics, *disableDeprecatedMetrics, *disableCinderAgentUUID, *domainID, *tenantID, novaMetadataMapping, nil, logger); err != nil {
		logger.Error("Failed to collect from cache", "err", err)
		cancel(err)
		return
	}

	for {
		select {
		case <-collectTicker.C:
			if err := cache.CollectCache(exporters.EnableExporter, *multiCloud, services, *prefix, *cloud, *disabledMetrics, *endpointType, *collectTime, *disableSlowMetrics, *disableDeprecatedMetrics, *disableCinderAgentUUID, *domainID, *tenantID, novaMetadataMapping, nil, logger); err != nil {
				cancel(err)
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

func startHTTPServer(ctx context.Context, services map[string]*bool, toolkitFlags *web.FlagConfig, cancel context.CancelCauseFunc, logger *slog.Logger) {
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
			cancel(err)
			return
		}
		http.Handle("/", landingPage)
	}

	if *domainID != "" {
		logger.Info("Gathering metrics for configured domain ID", "domain_id", *domainID)
	}

	if *tenantID != "" {
		logger.Info("Gathering metrics for configured tenant ID", "tenant_id", *tenantID)
	}

	srv := &http.Server{
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	go func() {
		if err := web.ListenAndServe(srv, toolkitFlags, logger); err != nil {
			logger.Error("Failed to start webserver", "error", err)
			cancel(err)
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
			exp, err := exporters.EnableExporter(service, *prefix, cloud, *disabledMetrics, *endpointType, *collectTime, *disableSlowMetrics, *disableDeprecatedMetrics, *disableCinderAgentUUID, *domainID, *tenantID, novaMetadataMapping, nil, logger)
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
			exp, err := exporters.EnableExporter(service, *prefix, *cloud, *disabledMetrics, *endpointType, *collectTime, *disableSlowMetrics, *disableDeprecatedMetrics, *disableCinderAgentUUID, *domainID, *tenantID, novaMetadataMapping, nil, logger)
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

		// expose program version
		registry.MustRegister(pver.NewCollector(progName))

		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}
