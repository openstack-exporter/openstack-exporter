package main

import (
	"fmt"
	"net/http"

	"github.com/creasty/defaults"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func EnableExporter(service string, prefix string, config *Cloud) (*OpenStackExporter, error) {
	// Set the default values for config structure.
	defaults.Set(config)

	exporter, err := NewExporter(service, prefix, config)
	if err != nil {
		return nil, err
	}
	prometheus.MustRegister(exporter)
	return &exporter, nil
}

var defaultEnabledServices = []string{"network", "compute", "image", "volumev3", "identity"}

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
				log.Fatal(err)
			}
			log.Infof("Enabled exporter for service: %s", service)
		}
	}

	http.Handle(*metrics, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>OpenStack Exporter</title></head>
             <body>
             <h1>OpenStack Exporter</h1>
             <p><a href='` + *metrics + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	log.Infoln("Starting HTTP server on", *bind)
	log.Fatal(http.ListenAndServe(*bind, nil))
}
