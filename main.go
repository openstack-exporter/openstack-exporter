package main

import (
	"fmt"
	"github.com/openstack-exporter/openstack-exporter/exporters"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

func EnableExporter(service, prefix, cloud string) (*exporters.OpenStackExporter, error) {
	exporter, err := exporters.NewExporter(service, prefix, cloud)
	if err != nil {
		return nil, err
	}
	prometheus.MustRegister(exporter)
	return &exporter, nil
}

var defaultEnabledServices = []string{"network", "compute", "image", "volume", "identity"}
var DEFAULT_OS_CLIENT_CONFIG = "/etc/openstack/clouds.yaml"

func main() {
	var (
		bind           = kingpin.Flag("web.listen-address", "address:port to listen on").Default(":9180").String()
		metrics        = kingpin.Flag("web.telemetry-path", "uri path to expose metrics").Default("/metrics").String()
		osClientConfig = kingpin.Flag("os-client-config", "Path to the cloud configuration file").Default(DEFAULT_OS_CLIENT_CONFIG).String()
		prefix         = kingpin.Flag("prefix", "Prefix for metrics").Default("openstack").String()
		cloud          = kingpin.Arg("cloud", "name or id of the cloud to gather metrics from").Required().String()
	)

	services := make(map[string]*bool)

	for _, service := range defaultEnabledServices {
		flagName := fmt.Sprintf("disable-service.%s", service)
		flagHelp := fmt.Sprintf("Disable the %s service exporter", service)
		services[service] = kingpin.Flag(flagName, flagHelp).Default().Bool()
	}

	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infof("Starting openstack exporter version %s for cloud: %s", version.Info(), *cloud)
	log.Infoln("Build context", version.BuildContext())

	if *osClientConfig != DEFAULT_OS_CLIENT_CONFIG {
		log.Debugf("Setting Env var OS_CLIENT_CONFIG_FILE = %s", *osClientConfig)
		os.Setenv("OS_CLIENT_CONFIG_FILE", *osClientConfig)
	}

	for service, disabled := range services {
		if !*disabled {
			_, err := EnableExporter(service, *prefix, *cloud)
			if err != nil {
				// Log error and continue with enabling other exporters
				log.Errorf("enabling exporter for service %s failed: %s", service, err)
				continue
			}
			log.Infof("Enabled exporter for service: %s", service)
		}
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

	log.Infoln("Starting HTTP server on", *bind)
	log.Fatal(http.ListenAndServe(*bind, nil))
}
