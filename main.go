package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	kingpin "github.com/alecthomas/kingpin/v2"
	clientconfigv2 "github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/jpillora/backoff"
	"gopkg.in/yaml.v3"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
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

const DEFAULT_OS_CLIENT_CONFIG = "/etc/openstack/clouds.yaml"

const (
	reloadRetryMinWait = 5 * time.Second
	reloadRetryMaxWait = 5 * time.Minute
)

type serviceState int

const (
	serviceAuto serviceState = iota
	serviceEnabled
	serviceDisabled
)

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
	tenantID                 = kingpin.Flag("project-id", "Gather metrics only for the given Project ID (defaults to all projects)").String()
	disableServiceAutodetect = kingpin.Flag("disable-service-autodetect", "Disable single-cloud service autodetection and use only explicit service flags").Default("false").Bool()
	novaMetadataMapping      = utils.LabelMapping(kingpin.Flag("nova.metadata-extra-labels", "Map provided server metadata keys to labels in openstack_nova_server_status metric").PlaceHolder("LABEL=KEY,KEY").Default(""))
	reloadInterval           = kingpin.Flag("reload-interval", "How often services are re-detected and every collector is re-bound; SIGHUP reloads at any time; 0 disables the timer").Default("1h").Duration()
	dnsConcurrentCount       = kingpin.Flag("dns-concurrent-count", "Number of concurrent requests for DNS recordset collection").Default("10").Int()
)

func main() {

	serviceStates := make(map[string]serviceState, len(exporters.SupportedExporters))

	for _, service := range exporters.SupportedExporters {
		serviceStates[service] = serviceAuto
		disableFlagName := fmt.Sprintf("disable-service.%s", service)
		disableFlagHelp := fmt.Sprintf("Disable the %s service exporter in strict mode", service)
		serviceName := service
		disableValue := false
		kingpin.Flag(disableFlagName, disableFlagHelp).Default("false").Action(func(*kingpin.ParseContext) error {
			if disableValue {
				serviceStates[serviceName] = serviceDisabled
			} else {
				serviceStates[serviceName] = serviceEnabled
			}
			return nil
		}).BoolVar(&disableValue)
	}
	toolkitFlags := webflag.AddFlags(kingpin.CommandLine, ":9180")

	promlogConfig := &promslog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("openstack-exporter"))
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

	SetPasswordIfVaultIsUsed(logger)

	if _, err := os.Stat(*osClientConfig); err != nil {
		logger.Error("Could not read config file", "error", err)
		os.Exit(1)
	}

	resolveParams := ResolveServiceConfigParams{
		MultiCloud:        *multiCloud,
		Cloud:             *cloud,
		DisableAutodetect: *disableServiceAutodetect,
		ServiceStates:     serviceStates,
	}

	// Without autodetection the service set comes from flags alone, so an
	// empty set is a configuration error worth failing on right away.
	var configuredServices []string
	if *multiCloud || *disableServiceAutodetect {
		var err error
		configuredServices, err = resolveServiceConfig(resolveParams, logger)
		if err != nil {
			logger.Error("Failed to resolve service configuration", "error", err)
			os.Exit(1)
		}
	}

	exporterOpts := exporters.ExporterOptions{
		Cloud:                    *cloud,
		Prefix:                   *prefix,
		DisabledMetrics:          *disabledMetrics,
		EndpointType:             *endpointType,
		CollectTime:              *collectTime,
		DisableSlowMetrics:       *disableSlowMetrics,
		DisableDeprecatedMetrics: *disableDeprecatedMetrics,
		DisableCinderAgentUUID:   *disableCinderAgentUUID,
		DomainID:                 *domainID,
		TenantID:                 *tenantID,
		NovaMetadataMapping:      novaMetadataMapping,
		DnsConcurrentCount:       *dnsConcurrentCount,
	}

	pool := exporters.NewPool(exporterOpts, logger)

	ctx1, cancel1 := context.WithCancelCause(context.Background())
	defer cancel1(nil)

	ctx2, cancel2 := signal.NotifyContext(ctx1, syscall.SIGINT, syscall.SIGTERM)
	defer cancel2()

	// Bind the collectors now and again on every reload, so the HTTP server
	// can come up before the first bind finishes and a service that is down
	// at startup is picked up later.
	var reload func(context.Context) error
	var currentServices func() []string
	detection := &serviceDetection{cloud: *cloud}
	if *multiCloud {
		reload = func(ctx context.Context) error {
			pool.ReloadAll(ctx)
			return nil
		}
		currentServices = func() []string { return configuredServices }
	} else {
		reload = func(ctx context.Context) error {
			return reloadSingleCloud(ctx, pool, resolveParams, detection, logger)
		}
		currentServices = func() []string { return pool.Services(*cloud) }
	}
	go runReloader(ctx2, *reloadInterval, reload, logger)

	// Start the backend service.
	if *cacheEnable {
		go cacheBackgroundService(ctx2, pool, currentServices, cancel1, logger)
	}

	// Start the HTTP server.
	go startHTTPServer(ctx2, StartHTTPServerParams{
		Pool:         pool,
		Services:     configuredServices,
		Detection:    detection,
		ToolkitFlags: toolkitFlags,
	}, cancel1, logger)

	<-ctx2.Done()
	if err := context.Cause(ctx2); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("Shutting down due to error", "error", err)
		os.Exit(1)
	} else {
		logger.Info("Termination signal received. Shutting down...")
	}
}

// ResolveServiceConfigParams holds the arguments for resolveServiceConfig.
type ResolveServiceConfigParams struct {
	MultiCloud        bool
	Cloud             string
	DisableAutodetect bool
	ServiceStates     map[string]serviceState
}

// resolveServiceConfig returns the services to collect. With autodetection it
// reads the Keystone service catalog, so it can fail and is called on every
// reload. The caller's ServiceStates map is left untouched.
func resolveServiceConfig(params ResolveServiceConfigParams, logger *slog.Logger) ([]string, error) {
	params.ServiceStates = maps.Clone(params.ServiceStates)
	if params.MultiCloud || params.DisableAutodetect {
		setAutoServicesState(params.ServiceStates, serviceEnabled)
		if params.DisableAutodetect && !params.MultiCloud {
			logger.Info("Service autodetection is disabled for single-cloud mode")
		}
		enabledServices := getEnabledServicesFromStates(params.ServiceStates)
		if len(enabledServices) == 0 {
			return nil, errors.New("no services enabled by explicit flags")
		}
		return enabledServices, nil
	}

	logger.Info("Autodetecting available services")
	detectedServices, err := autodetectServices(params.Cloud, logger)
	if err != nil {
		return nil, err
	}
	logger.Info("Autodetected services", "detected_services", detectedServices)

	applyAutodetection(params.ServiceStates, detectedServices)
	enabledServices := getEnabledServicesFromStates(params.ServiceStates)
	logger.Info("Final enabled services", "enabled_services", enabledServices)
	if len(enabledServices) == 0 {
		return nil, errors.New("no services enabled after autodetection and flag filtering")
	}

	return enabledServices, nil
}

func setAutoServicesState(serviceStates map[string]serviceState, state serviceState) {
	for _, service := range exporters.SupportedExporters {
		if serviceStates[service] == serviceAuto {
			serviceStates[service] = state
		}
	}
}

func applyAutodetection(serviceStates map[string]serviceState, detectedServices []string) {
	for _, service := range detectedServices {
		if serviceStates[service] == serviceAuto {
			serviceStates[service] = serviceEnabled
		}
	}
	setAutoServicesState(serviceStates, serviceDisabled)
}

func getEnabledServicesFromStates(serviceStates map[string]serviceState) []string {
	enabledServices := []string{}
	for _, service := range exporters.SupportedExporters {
		if serviceStates[service] == serviceEnabled {
			enabledServices = append(enabledServices, service)
		}
	}
	return enabledServices
}

func autodetectServices(cloud string, logger *slog.Logger) ([]string, error) {
	opts := &clientconfigv2.ClientOpts{Cloud: cloud}
	services, err := exporters.AutodetectServicesFromCatalog(opts, nil, *endpointType)
	if err != nil {
		return nil, err
	}
	return services, nil
}

// cacheBackgroundService runs a background service to collect the metrics and stores in the cache.
// It collects data every cache-ttl/2 time and flush every cache-ttl time.
// The cache data will be read by the Prometheus HandleFunc.
func cacheBackgroundService(ctx context.Context, pool *exporters.Pool, services func() []string, cancel context.CancelCauseFunc, logger *slog.Logger) {
	logger.Info("Start cache background service")
	collectTicker := time.NewTicker(*cacheTTL / 2)
	defer collectTicker.Stop()
	ttlTicker := time.NewTicker(*cacheTTL)
	defer ttlTicker.Stop()

	// Collect cache data in the beginning.
	if err := cache.CollectCache(ctx, pool, *multiCloud, services(), *cloud, logger); err != nil {
		logger.Error("Failed to collect from cache", "err", err)
		cancel(err)
		return
	}

	for {
		select {
		case <-collectTicker.C:
			if err := cache.CollectCache(ctx, pool, *multiCloud, services(), *cloud, logger); err != nil {
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

var serviceDetectionDesc = prometheus.NewDesc(
	"openstack_exporter_service_autodetect_success",
	"1 when the last lookup of the service catalog succeeded, 0 when the exporter runs on the previous or the explicitly enabled service set.",
	[]string{"cloud"}, nil,
)

// serviceDetection reports the outcome of the last service autodetection of
// the single cloud. Nothing is reported before the first attempt.
type serviceDetection struct {
	cloud string
	// state is 0 before the first attempt, 1 after a success, 2 after a failure.
	state atomic.Int32
}

func (d *serviceDetection) record(err error) {
	if err != nil {
		d.state.Store(2)
		return
	}
	d.state.Store(1)
}

func (d *serviceDetection) Describe(ch chan<- *prometheus.Desc) {
	ch <- serviceDetectionDesc
}

func (d *serviceDetection) Collect(ch chan<- prometheus.Metric) {
	switch d.state.Load() {
	case 1:
		ch <- prometheus.MustNewConstMetric(serviceDetectionDesc, prometheus.GaugeValue, 1, d.cloud)
	case 2:
		ch <- prometheus.MustNewConstMetric(serviceDetectionDesc, prometheus.GaugeValue, 0, d.cloud)
	}
}

// reloadSingleCloud re-detects the services of the configured cloud and
// rebinds them. When detection fails the previous service set is rebound
// instead, or just the explicitly enabled services if nothing was detected
// yet, and the error is returned so the caller retries sooner.
func reloadSingleCloud(ctx context.Context, pool *exporters.Pool, params ResolveServiceConfigParams, detection *serviceDetection, logger *slog.Logger) error {
	services, err := resolveServiceConfig(params, logger)
	detection.record(err)
	if err != nil {
		services = pool.Services(params.Cloud)
		if len(services) == 0 {
			services = getEnabledServicesFromStates(params.ServiceStates)
		}
		logger.Error("Failed to resolve service configuration, keeping the previous service set", "error", err, "services", services)
	}
	pool.Reload(ctx, params.Cloud, services)
	return err
}

// runReloader runs reload right away and again on every tick of interval and
// on SIGHUP. After a failed cycle the next one comes sooner, with backoff.
func runReloader(ctx context.Context, interval time.Duration, reload func(context.Context) error, logger *slog.Logger) {
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)

	retry := &backoff.Backoff{Min: reloadRetryMinWait, Max: reloadRetryMaxWait, Factor: 2, Jitter: true}
	wait := time.After(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-hup:
			logger.Info("SIGHUP received, reloading collectors")
		case <-wait:
		}

		err := reload(ctx)
		next := interval
		if err != nil {
			next = retry.Duration()
			if interval > 0 && next > interval {
				next = interval
			}
		} else {
			retry.Reset()
		}
		if next > 0 {
			wait = time.After(next)
		} else {
			wait = nil
		}
	}
}

// StartHTTPServerParams holds the arguments for startHTTPServer.
type StartHTTPServerParams struct {
	Pool *exporters.Pool
	// Services is the configured set for the multi-cloud probe handler.
	Services []string
	// Detection is reported by the single-cloud metrics handler.
	Detection    *serviceDetection
	ToolkitFlags *web.FlagConfig
}

func startHTTPServer(ctx context.Context, params StartHTTPServerParams, cancel context.CancelCauseFunc, logger *slog.Logger) {
	links := []web.LandingLinks{}

	if *multiCloud {
		http.HandleFunc("/probe", probeHandler(params.Pool, params.Services, logger))
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
		http.HandleFunc(*metrics, metricHandler(params.Pool, params.Detection, logger))
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
		logger.Info("Gathering metrics for configured project ID", "project_id", *tenantID)
	}

	srv := &http.Server{
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	go func() {
		if err := web.ListenAndServe(srv, params.ToolkitFlags, logger); err != nil {
			logger.Error("Failed to start webserver", "error", err)
			cancel(err)
		}
	}()

	<-ctx.Done()
	if err := srv.Shutdown(context.Background()); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}
}

func probeHandler(pool *exporters.Pool, configuredServices []string, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		r = r.WithContext(ctx)

		cloud := r.URL.Query().Get("cloud")
		if cloud == "" {
			http.Error(w, "'cloud' parameter is missing", http.StatusBadRequest)
			return
		}

		enabledServices, err := selectServicesForRequest(configuredServices, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		logger.Info("Enabled services", "enabled_services", enabledServices)

		// Get data from cache
		if *cacheEnable {
			if err := cache.WriteCacheToResponse(w, r, cloud, enabledServices, logger); err != nil {
				logger.Error("Write cache to response failed", "error", err)
			}
			return
		}

		serveCollectors(w, r, pool.Collectors(ctx, cloud, enabledServices))
	}
}

// metricHandler serves the collectors of the single configured cloud. The
// pool binds them in the background; a scrape retries services that are not
// bound yet and reports every service's state in
// openstack_exporter_collector_ready.
func metricHandler(pool *exporters.Pool, detection *serviceDetection, logger *slog.Logger) http.HandlerFunc {
	logger.Info("Starting openstack exporter version for cloud", "version", version.Info(), "cloud", *cloud)
	logger.Info("Build context", "build_context", version.BuildContext())

	// Get data from cache
	if *cacheEnable {
		return func(w http.ResponseWriter, r *http.Request) {
			if err := cache.WriteCacheToResponse(w, r, *cloud, pool.Services(*cloud), logger); err != nil {
				logger.Error("Write cache to response failed", "error", err)
			}
		}
	}

	// expose program version and the service detection state
	extra := []prometheus.Collector{pver.NewCollector("openstack_exporter"), detection}

	return func(w http.ResponseWriter, r *http.Request) {
		serveCollectors(w, r, pool.Collectors(r.Context(), *cloud, pool.Services(*cloud)), extra...)
	}
}

// serveCollectors answers a scrape from the bound collectors, their readiness
// gauges and any extra collectors.
func serveCollectors(w http.ResponseWriter, r *http.Request, bound exporters.Bound, extra ...prometheus.Collector) {
	registry := prometheus.NewPedanticRegistry()
	registry.MustRegister(bound.Ready)
	for _, exp := range bound.Exporters {
		registry.MustRegister(exp)
	}
	for _, c := range extra {
		registry.MustRegister(c)
	}

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func selectServicesForRequest(configuredServices []string, r *http.Request) ([]string, error) {
	enabledServices := configuredServices

	includeServices := r.URL.Query().Get("include_services")
	if includeServices != "" {
		includeList := parseServiceList(includeServices)
		invalid := invalidExporterNames(includeList)
		if len(invalid) > 0 {
			return nil, fmt.Errorf("invalid include_services: %s", strings.Join(invalid, ","))
		}
		enabledServices = includeList
	}

	excludeList := parseServiceList(r.URL.Query().Get("exclude_services"))
	invalid := invalidExporterNames(excludeList)
	if len(invalid) > 0 {
		return nil, fmt.Errorf("invalid exclude_services: %s", strings.Join(invalid, ","))
	}
	return utils.UniqueElements(utils.RemoveElements(enabledServices, excludeList)), nil
}

func invalidExporterNames(services []string) []string {
	invalid := make([]string, 0, len(services))
	for _, service := range services {
		if !exporters.IsExporterNameValid(service) {
			invalid = append(invalid, service)
		}
	}
	return invalid
}

func parseServiceList(raw string) []string {
	if raw == "" {
		return nil
	}

	services := []string{}
	for _, service := range strings.Split(raw, ",") {
		service = strings.TrimSpace(service)
		if service == "" {
			continue
		}
		services = append(services, service)
	}

	return services
}

func SetPasswordIfVaultIsUsed(logger *slog.Logger) {
	configFileData, err := os.ReadFile(*osClientConfig)
	if err != nil {
		logger.Error("failed to read config file", "err", err)
		os.Exit(1)
	}

	type VaultConfig struct {
		UseVault                    bool   `yaml:"use_vault"`
		VaultAddress                string `yaml:"vault_address"`
		VaultRoleID                 string `yaml:"vault_role_id"`
		VaultSecretID               string `yaml:"vault_secret_id"`
		VaultSecretPath             string `yaml:"vault_secret_path"`
		VaultSecretMountPath        string `yaml:"vault_secret_mount_path"`
		CredentialNameInVaultSecret string `yaml:"credential_name_in_vault_secret"`
	}

	var vaultConfig VaultConfig

	err = yaml.Unmarshal(configFileData, &vaultConfig)
	if err != nil {
		logger.Error("failed to parse config data", "err", err)
		os.Exit(1)
	}

	if !vaultConfig.UseVault {
		return
	}
	client, err := vault.New(vault.WithAddress(vaultConfig.VaultAddress))
	if err != nil {
		logger.Error("failed to create Vault client", "err", err)
		os.Exit(1)
	}
	ctx := context.Background()
	resp, err := client.Auth.AppRoleLogin(
		ctx,
		schema.AppRoleLoginRequest{
			RoleId:   vaultConfig.VaultRoleID,
			SecretId: vaultConfig.VaultSecretID,
		},
	)
	if err != nil {
		logger.Error("failed to login to Vault", "err", err)
		os.Exit(1)
	}
	if err := client.SetToken(resp.Auth.ClientToken); err != nil {
		logger.Error("failed to set Vault token", "err", err)
		os.Exit(1)
	}
	secret, err := client.Secrets.KvV2Read(
		ctx,
		vaultConfig.VaultSecretPath,
		vault.WithMountPath(vaultConfig.VaultSecretMountPath),
	)
	if err != nil {
		logger.Error("failed to get secret from Vault", "err", err)
		os.Exit(1)
	}

	os.Setenv("OS_PASSWORD", secret.Data.Data[vaultConfig.CredentialNameInVaultSecret].(string))
}
