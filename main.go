package main

import (
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/openstack-exporter/openstack-exporter/internal/config"
	"github.com/openstack-exporter/openstack-exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"net/http"
	"os"
	"strconv"
)

func main() {
	promlogConfig := &promlog.Config{}
	logger := promlog.New(promlogConfig)

	conf, err := config.New(logger)
	if err != nil {
		level.Error(logger).Log("msg", "opentelekomcloud-exporter: error: error loading config variables")
		os.Exit(1)
	}

	level.Info(logger).Log("msg", "config", "values", conf)
	if conf.Exporter.Cloud.Name == "" && !conf.Exporter.MultiCloud.IsEnabled {
		level.Error(logger).Log("msg", "opentelekomcloud-exporter: error: required argument 'cloud' or 'multi_cloud' not provided in config")
	}

	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())
	clientConfig := conf.Exporter.OSClientConfig
	if clientConfig != "" {
		level.Debug(logger).Log("msg", "setting environment var OS_CLIENT_CONFIG_FILE", "os_client_config_file", clientConfig)
		os.Setenv("OS_CLIENT_CONFIG_FILE", clientConfig)
	}

	var links []web.LandingLinks

	for key, api := range conf.Exporter.Api {
		level.Info(logger).Log("msg", "exporter started for "+api.Prefix.Name)
		http.HandleFunc(api.Metrics.Uri, metricHandler(key, api, conf.Exporter.Cloud.Name, logger))
		links = append(links, web.LandingLinks{
			Address: api.Metrics.Uri,
			Text:    api.Prefix.Name + " metrics",
		})
	}
	landingConfig := web.LandingConfig{
		Name:        "Exporter",
		Description: "Prometheus Exporter for Openstack/OpenTelekomCloud",
		Version:     version.Info(),
		Links:       links,
	}

	landingPage, err := web.NewLandingPage(landingConfig)
	if err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
	http.Handle("/", landingPage)

	srv := &http.Server{}
	srvFlag := web.FlagConfig{
		WebListenAddresses: utils.Pointer([]string{":" + strconv.Itoa(conf.Exporter.Port)}),
		WebSystemdSocket:   utils.Pointer(false),
		WebConfigFile:      utils.Pointer(""),
	}
	if err := web.ListenAndServe(srv, &srvFlag, logger); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
}

func metricHandler(name string, api config.ApiConfig, cloud string, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		level.Info(logger).Log("msg", "Starting apimon exporter version for cloud", "version", version.Info(), "cloud", cloud)
		level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())

		if cloud == "" {
			http.Error(w, "'cloud' parameter is missing", http.StatusBadRequest)
			return
		}

		registry := prometheus.NewPedanticRegistry()
		enabledExporters := 0
		for _, service := range api.Services {
			exp, err := exporters.EnableExporter(name, service, api.Prefix.Name, cloud, []string{}, api.EndpointType.Type, *api.CollectTime.IsEnabled, *api.Slow.Skip, *api.Deprecated.Skip, false, "", nil, logger)
			if err != nil {
				// Log error and continue with enabling other exporters
				level.Error(logger).Log("err", "enabling exporter for service failed", "service", service, "error", err)
				continue
			}
			registry.MustRegister(*exp)
			level.Info(logger).Log("msg", "enabled exporter for service", "service", service)
			enabledExporters++
		}

		if enabledExporters == 0 {
			level.Error(logger).Log("err", "no exporter has been enabled for API", "name", name)
		}

		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}
