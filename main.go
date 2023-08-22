package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

var defaultEnabledServices = []string{"network", "compute", "image", "volume", "identity", "object-store", "load-balancer", "container-infra", "dns", "baremetal", "gnocchi", "database", "orchestration", "placement"}

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
)

func main() {

	services := make(map[string]*bool)

	for _, service := range defaultEnabledServices {
		flagName := fmt.Sprintf("disable-service.%s", service)
		flagHelp := fmt.Sprintf("Disable the %s service exporter", service)
		services[service] = kingpin.Flag(flagName, flagHelp).Default().Bool()
	}
	toolkitFlags := webflag.AddFlags(kingpin.CommandLine, ":9180")

	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("openstack-exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	if *cloud == "" && !*multiCloud {
		level.Error(logger).Log("msg", "openstack-exporter: error: required argument 'cloud' or flag --multi-cloud not provided, try --help")
	}

	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())

	if *osClientConfig != DEFAULT_OS_CLIENT_CONFIG {
		level.Debug(logger).Log("msg", "Setting Env var OS_CLIENT_CONFIG_FILE", "os_client_config_file", *osClientConfig)
		os.Setenv("OS_CLIENT_CONFIG_FILE", *osClientConfig)
	}

	links := []web.LandingLinks{}

	if *multiCloud {
		http.HandleFunc("/probe", probeHandler(services, logger))
		http.Handle(*metrics, promhttp.Handler())
		level.Info(logger).Log("msg", "openstack exporter started in multi cloud mode (/probe?cloud=)")
		links = append(links, web.LandingLinks{
			Address: *metrics,
			Text:    "Metrics",
		}, web.LandingLinks{
			Address: "/probe",
			Text:    "Probes",
		})
	} else {
		level.Info(logger).Log("msg", "openstack exporter started in legacy mode")
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
			level.Error(logger).Log("err", err)
			os.Exit(1)
		}
		http.Handle("/", landingPage)
	}

	if *domainID != "" {
		level.Info(logger).Log("msg", "Gathering metrics for configured domain ID", "domain_id", *domainID)
	}

	srv := &http.Server{}
	if err := web.ListenAndServe(srv, toolkitFlags, logger); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
}

func probeHandler(services map[string]*bool, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		r = r.WithContext(ctx)

		registry := prometheus.NewPedanticRegistry()

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

		level.Info(logger).Log("msg", "Enabled services", "enabled_services", enabledServices)

		for _, service := range enabledServices {
			exp, err := exporters.EnableExporter(service, *prefix, cloud, *disabledMetrics, *endpointType, *collectTime, *disableSlowMetrics, *disableDeprecatedMetrics, *disableCinderAgentUUID, *domainID, nil, logger)
			if err != nil {
				level.Error(logger).Log("err", "Enabling exporter for service failed", "service", service, "error", err)
				continue
			}
			registry.MustRegister(*exp)
			level.Info(logger).Log("msg", "Enabled exporter for service", "service", service)
		}

		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}

func metricHandler(services map[string]*bool, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		level.Info(logger).Log("msg", "Starting openstack exporter version for cloud", "version", version.Info(), "cloud", *cloud)
		level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())

		if *osClientConfig != DEFAULT_OS_CLIENT_CONFIG {
			level.Debug(logger).Log("msg", "Setting Env var OS_CLIENT_CONFIG_FILE", "os_client_config_file", *osClientConfig)
			os.Setenv("OS_CLIENT_CONFIG_FILE", *osClientConfig)
		}

		registry := prometheus.NewPedanticRegistry()
		enabledExporters := 0
		for service, disabled := range services {
			if !*disabled {
				exp, err := exporters.EnableExporter(service, *prefix, *cloud, *disabledMetrics, *endpointType, *collectTime, *disableSlowMetrics, *disableDeprecatedMetrics, *disableCinderAgentUUID, *domainID, nil, logger)
				if err != nil {
					// Log error and continue with enabling other exporters
					level.Error(logger).Log("err", "enabling exporter for service failed", "service", service, "error", err)
					continue
				}
				registry.MustRegister(*exp)
				level.Info(logger).Log("msg", "Enabled exporter for service", "service", service)
				enabledExporters++
			}
		}

		if enabledExporters == 0 {
			level.Error(logger).Log("err", "No exporter has been enabled, exiting")
			os.Exit(-1)
		}

		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}
