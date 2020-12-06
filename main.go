package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

var defaultEnabledServices = []string{"network", "compute", "image", "volume", "identity", "object-store", "load-balancer", "container-infra", "dns", "baremetal", "gnocchi"}

var DEFAULT_OS_CLIENT_CONFIG = "/etc/openstack/clouds.yaml"

func main() {
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
		cloud              = kingpin.Arg("cloud", "name or id of the cloud to gather metrics from").Required().String()
	)

	services := make(map[string]*bool)

	for _, service := range defaultEnabledServices {
		flagName := fmt.Sprintf("disable-service.%s", service)
		flagHelp := fmt.Sprintf("Disable the %s service exporter", service)
		services[service] = kingpin.Flag(flagName, flagHelp).Default().Bool()
	}

	kingpin.Version(version.Print("openstack-exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	err := log.Base().SetLevel(*logLevel)
	if err != nil {
		log.Errorf("Cannot init set logger level: %s", err)
		os.Exit(-1)
	}

	log.Infof("Starting openstack exporter version %s for cloud: %s", version.Info(), *cloud)
	log.Infoln("Build context", version.BuildContext())

	if *osClientConfig != DEFAULT_OS_CLIENT_CONFIG {
		log.Debugf("Setting Env var OS_CLIENT_CONFIG_FILE = %s", *osClientConfig)
		os.Setenv("OS_CLIENT_CONFIG_FILE", *osClientConfig)
	}

	enabledExporters := 0
	for service, disabled := range services {
		if !*disabled {
			_, err := exporters.EnableExporter(service, *prefix, *cloud, *disabledMetrics, *endpointType, *collectTime, *disableSlowMetrics, nil)
			if err != nil {
				// Log error and continue with enabling other exporters
				log.Errorf("enabling exporter for service %s failed: %s", service, err)
				continue
			}
			log.Infof("Enabled exporter for service: %s", service)
			enabledExporters++
		}
	}

	if enabledExporters == 0 {
		log.Errorln("No exporter has been enabled, exiting")
		os.Exit(-1)
	}

	sm := http.NewServeMux()
	sm.Handle(*metrics, promhttp.Handler())
	sm.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

	tcp := ip4or6(*bind)

	l, err := net.Listen(tcp, *bind)
	if err != nil {
		log.Fatal(err)
	}

	log.Infoln("Starting HTTP server on", *bind)
	log.Fatal(http.Serve(l, sm))
}

func ip4or6(s string) string {
	re := regexp.MustCompile(":\\d*$")
	found := re.FindAllString(s, 1)
	s = strings.TrimSuffix(s, found[0])
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.':
			return "tcp4"
		case ':':
			return "tcp6"
		}
	}
	return "tcp"

}
