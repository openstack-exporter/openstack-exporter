# OpenStack Exporter for Prometheus

[![CI](https://github.com/openstack-exporter/openstack-exporter/actions/workflows/ci.yaml/badge.svg)](https://github.com/openstack-exporter/openstack-exporter/actions/workflows/ci.yaml)

A [OpenStack](https://openstack.org/) exporter for prometheus written in Golang using the
[gophercloud](https://github.com/gophercloud/gophercloud) library.

## Deployment options

The openstack-exporter can be deployed using the following mechanisms:

* By using [kolla-ansible](https://github.com/openstack/kolla-ansible) by setting enable_prometheus_openstack_exporter: true
* By using [helm charts](https://github.com/openstack-exporter/helm-charts)
* Via docker images, available from our [repository](https://github.com/openstack-exporter/openstack-exporter/pkgs/container/openstack-exporter)
* Via snaps from [snapcraft](https://snapcraft.io/golang-openstack-exporter)

### Latest Docker main images

Multi-arch images (amd64, arm64 and s390x)

```sh
docker pull ghcr.io/openstack-exporter/openstack-exporter:latest
```

### Release Docker images

Multi-arch images (amd64, arm64 and s390x)

```sh
docker pull ghcr.io/openstack-exporter/openstack-exporter:1.6.0
```

### Snaps

The exporter is also available on the [https://snapcraft.io/golang-openstack-exporter](https://snapcraft.io/golang-openstack-exporter)
For installing the latest master build (edge channel):

```sh
snap install --channel edge golang-openstack-exporter
```

For installing the latest stable version (stable channel):

```sh
snap install --channel stable golang-openstack-exporter
```

## Description

The OpenStack exporter, exports Prometheus metrics from a running OpenStack cloud
for consumption by prometheus. The cloud credentials and identity configuration
should use the [os-client-config](https://docs.openstack.org/os-client-config/latest/) format
and can be specified with the `--os-client-config` flag (defaults to `/etc/openstack/clouds.yaml`).

Other options as the binding address/port can by explored with the --help flag.

The exporter can operate in 2 modes

* A Legacy mode (targetting one cloud) in where the openstack\_exporter serves on port `0.0.0.0:9180` at the `/metrics` URL.
* A multi cloud mode in where the openstack\_exporter serves on port `0.0.0.0:9180` at the `/probe` URL.
  And where `/metrics` URL is serving own exporter metrics

You can build it by yourself by cloning this repository and run:

```sh
go build -o ./openstack-exporter .
```

Multi cloud mode

```sh
./openstack-exporter --os-client-config /etc/openstack/clouds.yaml --multi-cloud
curl "http://localhost:9180/probe?cloud=region.mycludprovider.org"
```

or Legacy mode

```sh
./openstack-exporter --os-client-config /etc/openstack/clouds.yaml myregion.cloud.org
curl "http://localhost:9180/metrics" +
```

Or alternatively you can use the docker images, as follows (check the openstack configuration section for configuration
details):

```sh
docker run -v "$HOME/.config/openstack/clouds.yml":/etc/openstack/clouds.yaml -it -p 9180:9180 \
ghcr.io/openstack-exporter/openstack-exporter:latest
curl "http://localhost:9180/probe?cloud=my-cloud.org"
```

### Command line options

The current list of command line options (by running --help)

```sh
usage: openstack-exporter [<flags>] [<cloud>]


Flags:
  -h, --[no-]help                Show context-sensitive help (also try --help-long and --help-man).
      --web.telemetry-path="/metrics"
                                 uri path to expose metrics
      --os-client-config="/etc/openstack/clouds.yaml"
                                 Path to the cloud configuration file
      --prefix="openstack"       Prefix for metrics
      --endpoint-type="public"   openstack endpoint type to use (i.e: public, internal, admin)
      --[no-]collect-metric-time
                                 Emit per-source fetch duration metrics
  -d, --disable-metric= ...      multiple --disable-metric can be specified in the format: exporter-metric (i.e: cinder-snapshots)
  -e, --enable-metric= ...       override disable-slow-metrics / disable-deprecated-metrics for individual metrics; format: exporter-metric (i.e: nova-limits_vcpus_max)
      --[no-]disable-slow-metrics
                                 Disable slow metrics for performance reasons
      --[no-]disable-deprecated-metrics
                                 Disable deprecated metrics
      --[no-]disable-cinder-agent-uuid
                                 Disable UUID generation for Cinder agents
      --[no-]multi-cloud         Toggle the multiple cloud scraping mode under /probe?cloud=
      --domain-id=DOMAIN-ID      Gather metrics only for the given Domain ID (defaults to all domains)
      --[no-]cache               Enable Cache mechanism globally
      --cache-ttl=300s           TTL duration for cache expiry(eg. 10s, 11m, 1h)
      --project-id=PROJECT-ID    Gather metrics only for the given Project ID (defaults to all projects)
      --[no-]disable-service-autodetect
                                 Disable single-cloud service autodetection and use only explicit service flags
      --nova.metadata-extra-labels=LABEL=KEY,KEY ...
                                 Map provided server metadata keys to labels in openstack_nova_server_status metric
      --dns-concurrent-count=10  Number of concurrent requests for DNS recordset collection
      --placement-concurrent-count=10
                                 Number of concurrent requests for Placement provider detail collection
      --[no-]disable-service.network
                                 Disable the network service exporter in strict mode
      --[no-]disable-service.compute
                                 Disable the compute service exporter in strict mode
      --[no-]disable-service.image
                                 Disable the image service exporter in strict mode
      --[no-]disable-service.volume
                                 Disable the volume service exporter in strict mode
      --[no-]disable-service.identity
                                 Disable the identity service exporter in strict mode
      --[no-]disable-service.object-store
                                 Disable the object-store service exporter in strict mode
      --[no-]disable-service.load-balancer
                                 Disable the load-balancer service exporter in strict mode
      --[no-]disable-service.container-infra
                                 Disable the container-infra service exporter in strict mode
      --[no-]disable-service.dns
                                 Disable the dns service exporter in strict mode
      --[no-]disable-service.baremetal
                                 Disable the baremetal service exporter in strict mode
      --[no-]disable-service.gnocchi
                                 Disable the gnocchi service exporter in strict mode
      --[no-]disable-service.database
                                 Disable the database service exporter in strict mode
      --[no-]disable-service.orchestration
                                 Disable the orchestration service exporter in strict mode
      --[no-]disable-service.placement
                                 Disable the placement service exporter in strict mode
      --[no-]disable-service.sharev2
                                 Disable the sharev2 service exporter in strict mode
      --[no-]web.systemd-socket  Use systemd socket activation listeners instead of port listeners (Linux only).
      --web.listen-address=:9180 ...
                                 Addresses on which to expose metrics and web interface. Repeatable for multiple addresses. Examples: `:9100` or `[::1]:9100` for http, `vsock://:9100` for vsock
      --web.config.file=""       Path to configuration file that can enable TLS or authentication. See: https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md
      --log.level=info           Only log messages with the given severity or above. One of: [debug, info, warn, error]
      --log.format=logfmt        Output format of log messages. One of: [logfmt, json]
      --[no-]version             Show application version.

Args:
  [<cloud>]  name or id of the cloud to gather metrics from
```

### Scrape options

In legacy mode cloud and metrics to be scraped are specified as argument or flags as described above.
To select multi cloud mode the --multi-cloud flag needs to be used.
In that case metrics and clouds are specified in the http scrape request as described below.
Which cloud (name or id from the `clouds.yaml` file) or what services from the cloud to scrape, can be specified as the parameters to http scrape requests.

Single-cloud service selection behavior:

* By default, services are auto-detected from Keystone service catalog.
* `--disable-service.<service>` explicitly disables that exporter.
* `--no-disable-service.<service>` explicitly enables that exporter.
* `--disable-service-autodetect` disables autodetection and treats remaining `AUTO` services as enabled (strict list mode from flags only).
* In `--multi-cloud` mode, autodetection is not used; service flags define the configured set used by `/probe`.

Query Parameter | Description
--- | ---
`cloud` | Name or id of the cloud to gather metrics from (as specified in the `clouds.yaml`)
`include_services` | A comma separated list of services for which metrics will be scraped. It overrides the configured service set for that request.
`exclude_services` | A comma separated list of services for which metrics will *not* be scraped. Default is empty: ""

#### Examples

Scrape all services from `test.cloud`:

```sh
curl "https://localhost:9180/probe?cloud=test.cloud"
```

Scrape only `network` and `compute` services from `test.cloud`:

```sh
curl "https://localhost:9180/probe?cloud=test.cloud&include_services=network,compute"
```

Scrape all services except `load-balancer` and `dns` from `test.cloud`:

```sh
curl "https://localhost:9180/probe?cloud=test.cloud&exclude_services=load-balancer,dns"
```

### OpenStack configuration

The cloud credentials and identity configuration
should use the [os-client-config](https://docs.openstack.org/os-client-config/latest/) format
and can be specified with the `--os-client-config` flag (defaults to `/etc/openstack/clouds.yaml`).

`cacert` can be a full path to a PEM encoded file or contents of a PEM encoded file.

Current list of supported options can be seen in the following example
configuration:

```yaml
clouds:
  default:
    region_name: {{ openstack_region_name }}
    identity_api_version: 3
    identity_interface: internal
    auth:
      username: {{ keystone_admin_user }}
      password: {{ keystone_admin_password }}
      project_name: {{ keystone_admin_project }}
      project_domain_name: 'Default'
      project_domain_id: 'Default' // This can replace "project_domain_name"
      user_domain_name: 'Default'
      auth_url: {{ admin_protocol }}://{{ kolla_internal_fqdn }}:{{ keystone_admin_port }}/v3
    cacert: |
      ---- BEGIN CERTIFICATE ---
      ...
    verify: true | false  // disable || enable SSL certificate verification
```

#### Vault password lookup

If the same configuration file contains `use_vault: true`, the exporter logs
in to Vault with AppRole, reads a KV v2 secret, and sets `OS_PASSWORD` from one
field in that secret before creating OpenStack clients. The Vault settings are
top-level YAML keys, not entries under `clouds`.

```yaml
use_vault: true
vault_address: https://vault.example.org:8200
vault_role_id: {{ vault_role_id }}
vault_secret_id: {{ vault_secret_id }}
vault_secret_mount_path: secret
vault_secret_path: openstack/exporter
credential_name_in_vault_secret: password
```

The selected cloud should omit the inline password or otherwise allow the
OpenStack client config to use `OS_PASSWORD`.

### OpenStack Domain filtering

The exporter provides the flag `--domain-id`, this restricts some metrics to a specific domain.

*Restricting domain scope can improve scrape time, especially if you use Heat a lot.*

The following metrics are filtered for the domain ID provided (the others remain the same):

#### Cinder

* `openstack_cinder_limits_volume_max_gb`
* `openstack_cinder_limits_volume_used_gb`
* `openstack_cinder_limits_backup_max_gb`
* `openstack_cinder_limits_backup_used_gb`

#### Keystone

* `openstack_identity_projects`
* `openstack_identity_project_info`

#### Nova

* `openstack_nova_limits_vcpus_max`
* `openstack_nova_limits_vcpus_used`
* `openstack_nova_limits_memory_max`
* `openstack_nova_limits_memory_used`
* `openstack_nova_limits_instances_max`
* `openstack_nova_limits_instances_used`

### Cache mechanism

Enabling the cache with `--cache` changes the exporter's metric collection and delivery:

#### Background Service

* Collects metrics at the start and subsequently every half cache TTL.
* Updates the cache backend after completing each collection cycle.
* Flushes expired cache data every cache TTL.

#### Exporter API

* Returns no data if the cache is empty or expired.
* Retrieves and returns cached data from the backend.

## Contributing

Please file pull requests or issues under GitHub. Feel free to request any metrics
that might be missing.

Exporter internals, DAG execution, and the new-exporter checklist are documented
in [docs/exporter.md](docs/exporter.md).

### Operational Concerns

#### OpenStack Exporter Compatibility with Older OpenStack Versions

OpenStack evolves with new features and fields added to the API over time, leveraging a system of microversions to allow for incremental changes while maintaining backward compatibility. However, not all OpenStack deployments will be running the latest microversions, meaning certain fields or metrics may not be available when using an older version of OpenStack. Similarly, as newer OpenStack versions are adopted with more recent microversions, these changes are also seamlessly handled.

OpenStack-Exporter is designed to handle both older and newer microversions gracefully by default. When querying an OpenStack environment, it ensures compatibility with various microversions, using fallback behaviors for missing fields introduced in newer microversions:

* For Boolean Fields: Assumes the default value of false.
* For Numeric Fields: Missing numeric fields will default to 0
* For String Fields: Typically default to an empty string ("")

This fallback mechanism ensures that OpenStack-Exporter works correctly even when interfacing with OpenStack environments using older microversions, without causing operational disruptions.

## Metrics

The generated metrics inventory is available in [docs/metrics.md](docs/metrics.md).
It includes collected metric names, variable labels, slow/deprecated notes,
exporter-metric keys used by `--disable-metric` / `--enable-metric`, and
fixture-backed Prometheus sample lines.

Regenerate it after changing metric descriptors:

```sh
go run ./script/generate-metrics-doc.go
```

Check that the generated inventory is current:

```sh
go run ./script/generate-metrics-doc.go -check
```

## Cinder Volume Status Description

Index | Status
------|-------
0 |creating
1 |available
2 |reserved
3 |attaching
4 |detaching
5 |in-use
6 |maintenance
7 |deleting
8 |awaiting-transfer
9 |error
10 |error_deleting
11 |backing-up
12 |restoring-backup
13 |error_backing-up
14 |error_restoring
15 |error_extending
16 |downloading
17 |uploading
18 |retyping
19 |extending

## Manila Share Status Description

Index | Status
------|-------
0 |creating
1 |available
2 |updating
3 |migrating
4 |migration_error
5 |extending
6 |deleting
7 |shrinking
8 |error
9 |error_deleting
10 |shrinking_error
11 |reverting_error
12 |restoring-backup
13 |restoring
14 |reverting
15 |managing
16 |unmanaging
17 |reverting_to_snapshot
18 |soft_deleting
19 |inactive

### Communication

Please join us at #openstack-exporter at [OFTC](https://www.oftc.net/)
