package main

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func EnableExporter(service string, prefix string, config *Cloud) (*OpenStackExporter, error) {
	exporter, err := NewExporter(service, prefix, config)
	if err != nil {
		return nil, err
	}
	prometheus.MustRegister(exporter)
	return &exporter, nil
}

var defaultEnabledServices = []string{"identity", "image", "compute", "network", "volumev3"}

func main() {
	var (
		bind           = kingpin.Flag("web.listen-address", "address:port to listen on").Default(":9180").String()
		metrics        = kingpin.Flag("web.telemetry-path", "uri path to expose metrics").Default("/metrics").String()
		osClientConfig = kingpin.Flag("os-client-config", "Path to the cloud configuration file").Default("/etc/openstack/clouds.yml").String()
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

	log.Infoln("Starting openstack exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	config, err := NewCloudConfigFromFile(*osClientConfig)
	if err != nil {
		log.Fatal(err)
	}

	cloudConfig, err := config.GetByName(*cloud)
	if err != nil {
		log.Fatal(err)
	}

	for service, disabled := range services {
		if !*disabled {
			_, err := EnableExporter(service, *prefix, cloudConfig)
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
