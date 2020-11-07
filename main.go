package main

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

var defaultEnabledServices = []string{"network", "compute", "image", "volume", "identity", "object-store", "load-balancer", "container-infra", "dns", "baremetal", "gnocchi"}

var DEFAULT_OS_CLIENT_CONFIG = "/etc/openstack/clouds.yaml"

var (
	logLevel           = kingpin.Flag("log.level", "Log level: [debug, info, warn, error, fatal]").Default("info").String()
	bind               = kingpin.Flag("web.listen-address", "address:port to listen on").Default(":9180").String()
	metrics            = kingpin.Flag("web.telemetry-path", "uri path to expose metrics").Default("/metrics").String()
	osClientConfig     = kingpin.Flag("os-client-config", "Path to the cloud configuration file").Default(DEFAULT_OS_CLIENT_CONFIG).String()
	prefix             = kingpin.Flag("prefix", "Prefix for metrics").Default("openstack").String()
	endpointType       = kingpin.Flag("endpoint-type", "openstack endpoint type to use (i.e: public, internal, admin)").Default("public").String()
	collectTime        = kingpin.Flag("collect-metric-time", "time spent collecting each metric").Default("false").Bool()
	disabledMetrics    = kingpin.Flag("disable-metric", "multiple --disable-metric can be specified in the format: service-metric (i.e: cinder-snapshots)").Default("").Short('d').Strings()
	disableSlowMetrics = kingpin.Flag("disable-slow-metrics", "disable slow metrics for performance reasons").Default("false").Bool()
)

func main() {

	kingpin.Version(version.Print("openstack-exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	err := log.Base().SetLevel(*logLevel)
	if err != nil {
		log.Errorf("Cannot init set logger level: %s", err)
		os.Exit(-1)
	}

	log.Infoln("Build context", version.BuildContext())

	if *osClientConfig != DEFAULT_OS_CLIENT_CONFIG {
		log.Debugf("Setting Env var OS_CLIENT_CONFIG_FILE = %s", *osClientConfig)
		os.Setenv("OS_CLIENT_CONFIG_FILE", *osClientConfig)
	}

	http.Handle(*metrics, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`<html>
             <head><title>OpenStack Exporter</title></head>
             <body>
             <h1>OpenStack Exporter</h1>
             <p><a href='` + *metrics + `'>Metrics</a></p>
             </body>
             </html>`))
		if err != nil {
			log.Error(err)
		}
	})
	http.HandleFunc("/probe", probeHandler)

	log.Infoln("Starting HTTP server on", *bind)
	log.Fatal(http.ListenAndServe(*bind, nil))
}

func probeHandler(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	r = r.WithContext(ctx)

	registry := prometheus.NewPedanticRegistry()

	cloud := r.URL.Query().Get("cloud")
	if cloud == "" {
		http.Error(w, "'cloud' parameter is missing", http.StatusBadRequest)
		return
	}

	services := defaultEnabledServices
	includeServices := r.URL.Query().Get("include_services")
	if includeServices != "" {
		services = strings.Split(includeServices, ",")
	}

	excludeServices := strings.Split(r.URL.Query().Get("exclude_services"), ",")
	services = removeElements(services, excludeServices)

	log.Infof("Enabled services: %v", services)

	for _, service := range services {
		exp, err := exporters.EnableExporter(service, *prefix, cloud, *disabledMetrics, *endpointType, *collectTime, *disableSlowMetrics, nil)
		if err != nil {
			log.Errorf("enabling exporter for service %s failed: %s", service, err)
			continue
		}
		registry.MustRegister(*exp)
		log.Infof("Enabled exporter for service: %s", service)
	}

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func removeElements(slice []string, drop []string) []string {
	res := []string{}
	for _, s := range slice {
		keep := true
		for _, d := range drop {
			if s == d {
				keep = false
				break
			}
		}
		if keep {
			res = append(res, s)
		}
	}
	return res
}
