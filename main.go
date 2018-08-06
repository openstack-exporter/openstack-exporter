package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
)

func EnableExporter(service string, config *Cloud) (*OpenStackExporter, error) {
	exporter, err := NewExporter(service, config)
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
		osClientConfig = kingpin.Flag("os-client-config", "Path to the cloud configuration file").Default("/home/niedbalski/.config/openstack/clouds.yml").String()
		cloud          = kingpin.Arg("cloud", "name or id of the cloud to gather metrics from").Required().String()
	)

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
		_, err := EnableExporter(service, cloudConfig)
		if err != nil {
			panic(err)
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
