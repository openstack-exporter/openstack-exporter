package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
)

func EnableExporter(service string, prefix string, config *Cloud) (*OpenStackExporter, error) {
	exporter, err := NewExporter(service, prefix, config)
	if err != nil {
		return nil, err
	}
	prometheus.MustRegister(exporter)
	return &exporter, nil
}

var enabledServices = []string{"network", "compute", "image", "volume"}

func main() {
	var (
		bind           = kingpin.Flag("web.listen-address", "address:port to listen on").Default(":9180").String()
		metrics        = kingpin.Flag("web.telemetry-path", "uri path to expose metrics").Default("/metrics").String()
		osClientConfig = kingpin.Flag("os-client-config", "Path to the cloud configuration file").Default("/etc/openstack/clouds.yml").String()
		prefix         = kingpin.Flag("prefix", "Prefix for metrics").Default("openstack").String()
		cloud          = kingpin.Arg("cloud", "name or id of the cloud to gather metrics from").Required().String()
	)

	log.Infoln("Starting openstack exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	config, err := NewCloudConfigFromFile(*osClientConfig)
	if err != nil {
		panic(err)
	}

	cloudConfig, err := config.GetByName(*cloud)
	if err != nil {
		panic(err)
	}

	for _, service := range enabledServices {
		_, err := EnableExporter(service, *prefix, cloudConfig)
		if err != nil {
			panic(err)
		}
		log.Infoln("Enabled exporter for", service)
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
